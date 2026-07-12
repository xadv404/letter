package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xadv404/letter/internal/config"
	"github.com/xadv404/letter/internal/crawler"
	"github.com/xadv404/letter/internal/monitor"
)

// App is the HTML dashboard backend.
type App struct {
	mu         sync.Mutex
	cfg        config.CrawlConfig
	domainFile string
	engine     *crawler.Engine
	cancel     context.CancelFunc
	running    bool
	hub        *Hub
	logs       []string
	lastSnap   monitor.UISnapshot
	dorksPath  string
}

func newApp() *App {
	cfg := config.Default()
	return &App{
		cfg:        cfg,
		domainFile: "",
		hub:        NewHub(),
		logs:       []string{},
	}
}

type configDTO struct {
	DomainFile    string `json:"domain_file"`
	OutputDir     string `json:"output_dir"`
	Depth         int    `json:"depth"`
	PageLimit     int    `json:"page_limit"`
	Workers       int    `json:"workers"`
	DelayMS       int    `json:"delay_ms"`
	MinParamScore int    `json:"min_param_score"`
}

type snapshotDTO struct {
	Running     bool                   `json:"running"`
	Phase       int                    `json:"phase"`
	PhaseLabel  string                 `json:"phase_label"`
	Elapsed     string                 `json:"elapsed"`
	CPU         float64                `json:"cpu"`
	RAM         float64                `json:"ram"`
	Throttle    string                 `json:"throttle"`
	Workers     int                    `json:"workers"`
	DelayMS     int                    `json:"delay_ms"`
	Keywords    int                    `json:"keywords"`
	Params      int                    `json:"params"`
	Accepted    int                    `json:"accepted"`
	Rejected    int                    `json:"rejected"`
	Domains     []monitor.DomainStatus `json:"domains"`
	Decisions   []decisionDTO          `json:"decisions"`
	DorkPreview string                 `json:"dork_preview"`
	DorksPath   string                 `json:"dorks_path"`
	DomainFile  string                 `json:"domain_file"`
}

type decisionDTO struct {
	Param    string `json:"param"`
	Score    int    `json:"score"`
	Tier     string `json:"tier"`
	Reason   string `json:"reason"`
	Accepted bool   `json:"accepted"`
}

func (a *App) appendLog(msg string) {
	line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	a.logs = append(a.logs, line)
	if len(a.logs) > 200 {
		a.logs = a.logs[len(a.logs)-200:]
	}
	a.hub.Broadcast(mustJSON(map[string]string{"type": "log", "line": line}))
}

func (a *App) broadcastSnapshot(snap monitor.UISnapshot) {
	a.lastSnap = snap
	dto := a.toSnapshotDTO(snap)
	a.hub.Broadcast(mustJSON(map[string]any{"type": "snapshot", "data": dto}))
}

func (a *App) toSnapshotDTO(snap monitor.UISnapshot) snapshotDTO {
	decs := make([]decisionDTO, 0, len(snap.Decisions))
	start := 0
	if len(snap.Decisions) > 20 {
		start = len(snap.Decisions) - 20
	}
	for _, d := range snap.Decisions[start:] {
		decs = append(decs, decisionDTO{
			Param: d.Param, Score: d.Score, Tier: string(d.Tier),
			Reason: d.Reason, Accepted: d.Accepted,
		})
	}
	return snapshotDTO{
		Running: snap.Running, Phase: snap.Phase, PhaseLabel: snap.PhaseLabel,
		Elapsed: snap.Elapsed.String(), CPU: snap.CPU, RAM: snap.RAM,
		Throttle: snap.Throttle, Workers: snap.Workers, DelayMS: snap.DelayMS,
		Keywords: snap.Keywords, Params: snap.Params,
		Accepted: snap.Accepted, Rejected: snap.Rejected,
		Domains: snap.Domains, Decisions: decs,
		DorkPreview: snap.DorkPreview, DorksPath: a.dorksPath,
		DomainFile: a.domainFile,
	}
}

func (a *App) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	writeJSON(w, a.configResponse())
}

func (a *App) handleConfigPut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in configDTO
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running {
		http.Error(w, "cannot change config while running", http.StatusConflict)
		return
	}
	a.applyConfigDTO(in)
	writeJSON(w, a.configResponse())
}

func (a *App) applyConfigDTO(in configDTO) {
	if in.OutputDir != "" {
		a.cfg.OutputDir = in.OutputDir
	}
	if in.Depth > 0 {
		a.cfg.Depth = in.Depth
	}
	if in.PageLimit > 0 {
		a.cfg.PageLimit = in.PageLimit
	}
	if in.Workers > 0 {
		a.cfg.Workers = in.Workers
	}
	if in.DelayMS > 0 {
		a.cfg.DelayMS = in.DelayMS
	}
	if in.MinParamScore > 0 {
		a.cfg.MinParamScore = in.MinParamScore
	}
	if in.DomainFile != "" {
		a.domainFile = in.DomainFile
		a.cfg.DomainFile = in.DomainFile
	}
	a.cfg.StateFile = filepath.Join(a.cfg.OutputDir, "crawl.state.json")
}

func (a *App) configResponse() configDTO {
	return configDTO{
		DomainFile: a.domainFile, OutputDir: a.cfg.OutputDir,
		Depth: a.cfg.Depth, PageLimit: a.cfg.PageLimit,
		Workers: a.cfg.Workers, DelayMS: a.cfg.DelayMS,
		MinParamScore: a.cfg.MinParamScore,
	}
}

func (a *App) handleDomainsUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running {
		http.Error(w, "cannot upload while running", http.StatusConflict)
		return
	}
	if err := os.MkdirAll(a.cfg.OutputDir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dest := filepath.Join(a.cfg.OutputDir, "domains.txt")
	out, err := os.Create(dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out.Close()
	a.domainFile = dest
	a.cfg.DomainFile = dest
	a.appendLog("Domaines importés → " + dest)
	writeJSON(w, map[string]string{"domain_file": dest})
}

func (a *App) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		http.Error(w, "already running", http.StatusConflict)
		return
	}
	cfg := a.cfg
	cfg.DomainFile = a.domainFile
	cfg.StateFile = filepath.Join(cfg.OutputDir, "crawl.state.json")
	if cfg.DomainFile == "" {
		a.mu.Unlock()
		http.Error(w, "import a domains file first", http.StatusBadRequest)
		return
	}
	if err := cfg.Validate(); err != nil {
		a.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	domains, err := crawler.LoadDomains(cfg.DomainFile)
	if err != nil {
		a.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.dorksPath = ""
	a.running = true
	a.mu.Unlock()

	events := crawler.Events{
		OnSnapshot: func(s monitor.UISnapshot) {
			a.mu.Lock()
			s.Running = a.running
			a.broadcastSnapshot(s)
			a.mu.Unlock()
		},
		OnLog: func(msg string) {
			a.mu.Lock()
			a.appendLog(msg)
			a.mu.Unlock()
		},
		OnDorksDone: func(path string) {
			a.mu.Lock()
			a.dorksPath = path
			a.appendLog("Dorks prêtes → " + path)
			a.hub.Broadcast(mustJSON(map[string]string{"type": "dorks_ready", "path": path}))
			a.mu.Unlock()
		},
	}

	engine, err := crawler.NewWithEvents(cfg, events)
	if err != nil {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.engine = engine
	a.cancel = cancel
	a.mu.Unlock()

	a.appendLog(fmt.Sprintf("Démarrage sur %d domaine(s)…", len(domains)))

	go func() {
		err := engine.Run(ctx, domains)
		a.mu.Lock()
		a.running = false
		a.engine = nil
		a.cancel = nil
		if err != nil {
			a.appendLog("Erreur: " + err.Error())
		} else {
			a.appendLog("Recon terminée")
		}
		a.mu.Unlock()
	}()

	writeJSON(w, map[string]string{"status": "started"})
}

func (a *App) handlePause(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	eng := a.engine
	a.mu.Unlock()
	if eng == nil {
		http.Error(w, "not running", http.StatusBadRequest)
		return
	}
	eng.Pause()
	a.appendLog("Pause")
	writeJSON(w, map[string]string{"status": "paused"})
}

func (a *App) handleResume(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	eng := a.engine
	a.mu.Unlock()
	if eng == nil {
		http.Error(w, "not running", http.StatusBadRequest)
		return
	}
	eng.Resume()
	a.appendLog("Reprise")
	writeJSON(w, map[string]string{"status": "resumed"})
}

func (a *App) handleStop(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	eng := a.engine
	cancel := a.cancel
	a.mu.Unlock()
	if eng != nil {
		eng.Stop()
	}
	if cancel != nil {
		cancel()
	}
	a.appendLog("Arrêt demandé…")
	writeJSON(w, map[string]string{"status": "stopping"})
}

func (a *App) handleState(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	snap := a.lastSnap
	snap.Running = a.running
	dto := a.toSnapshotDTO(snap)
	writeJSON(w, map[string]any{
		"snapshot": dto,
		"config":   a.configResponse(),
		"logs":     append([]string(nil), a.logs...),
	})
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := a.hub.Subscribe()
	defer a.hub.Unsubscribe(ch)

	a.mu.Lock()
	init := map[string]any{
		"type": "init",
		"data": map[string]any{
			"snapshot": a.toSnapshotDTO(a.lastSnap),
			"config":   a.configResponse(),
			"logs":     append([]string(nil), a.logs...),
		},
	}
	a.mu.Unlock()
	fmt.Fprintf(w, "data: %s\n\n", mustJSON(init))
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (a *App) handleDorks(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	path := a.dorksPath
	if path == "" {
		path = filepath.Join(a.cfg.OutputDir, "dorks.txt")
	}
	a.mu.Unlock()
	if _, err := os.Stat(path); err != nil {
		http.Error(w, "dorks not ready", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="dorks.txt"`)
	http.ServeFile(w, r, path)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{}`
	}
	return string(b)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
