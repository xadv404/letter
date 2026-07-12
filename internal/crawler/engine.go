package crawler

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/html"

	"github.com/xadv404/letter/internal/config"
	"github.com/xadv404/letter/internal/dorks"
	"github.com/xadv404/letter/internal/export"
	"github.com/xadv404/letter/internal/keywords"
	"github.com/xadv404/letter/internal/monitor"
	"github.com/xadv404/letter/internal/params"
	"github.com/xadv404/letter/internal/state"
	"github.com/xadv404/letter/internal/throttle"
)

type pageRecord struct {
	host  string
	url   string
	body  string
	links []string
}

type Engine struct {
	cfg       config.CrawlConfig
	throttle  *throttle.Controller
	exporter  *export.Writer
	state     *state.Store
	dashboard *monitor.Dashboard
	kw        *keywords.Extractor
	scorer    *params.Scorer
	dorks     *dorks.Generator
	client    *http.Client

	paused   atomic.Bool
	active   atomic.Int32
	stopCh   chan struct{}
	stopOnce sync.Once

	pageBuf   []pageRecord
	pageBufMu sync.Mutex

	events Events
	phase  atomic.Int32
}

func New(cfg config.CrawlConfig) (*Engine, error) {
	return NewWithEvents(cfg, Events{})
}

func NewWithEvents(cfg config.CrawlConfig, events Events) (*Engine, error) {
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, err
	}
	st, err := state.Load(cfg.StateFile)
	if err != nil {
		return nil, err
	}
	resume := shouldResumeExport(st, cfg.DomainFile)
	exp, err := export.New(cfg.OutputDir, resume)
	if err != nil {
		return nil, err
	}
	e := &Engine{
		cfg:            cfg,
		throttle:       throttle.New(cfg.DelayMS, cfg.Depth, cfg.PageLimit, cfg.Workers),
		exporter:       exp,
		state:          st,
		dashboard:      monitor.New(),
		kw:             keywords.New(config.MaxKeywords),
		scorer:         params.New(config.MaxParams, cfg.OutputDir),
		dorks:          dorks.New(),
		client:         &http.Client{Timeout: 15 * time.Second},
		stopCh:         make(chan struct{}),
		events:         events,
	}
	if st.IsPaused() {
		e.paused.Store(true)
	}
	return e, nil
}

func shouldResumeExport(st *state.Store, domainFile string) bool {
	if st.IsPaused() {
		return true
	}
	if domainFile == "" {
		return false
	}
	domains, err := LoadDomains(domainFile)
	if err != nil {
		return false
	}
	for _, d := range domains {
		u, err := url.Parse(d)
		if err != nil || u.Host == "" {
			continue
		}
		host := NormalizeHost(u.Host)
		p := st.Get(host)
		if !p.Finished && (p.Pages > 0 || len(p.Queue) > 0 || len(p.Visited) > 0) {
			return true
		}
	}
	return false
}

func (e *Engine) log(msg string) {
	if e.events.OnLog != nil {
		e.events.OnLog(msg)
	} else {
		fmt.Println(msg)
	}
}

func (e *Engine) setPhase(n int, label string) {
	e.phase.Store(int32(n))
	e.emitSnapshot(label, n == 4, "")
}

func (e *Engine) emitSnapshot(phaseLabel string, running bool, dorkPreview string) {
	if e.events.OnSnapshot == nil {
		return
	}
	snap := e.throttle.Refresh(countGoroutines())
	acc, rej := e.scorer.Stats()
	kw := len(e.kw.Top(config.MaxExportKeywords))
	pm := e.scorer.Count()
	e.dashboard.RecordSeries(kw, pm)
	ui := e.dashboard.Snapshot(
		int(e.phase.Load()),
		phaseLabel,
		snap,
		e.scorer.Decisions(),
		acc, rej,
		dorkPreview,
		running,
	)
	ui.Keywords = kw
	ui.Params = pm
	e.events.OnSnapshot(ui)
}

func LoadDomains(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var domains []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "http") {
			line = "https://" + line
		}
		domains = append(domains, line)
	}
	return domains, sc.Err()
}

func (e *Engine) Pause() {
	e.paused.Store(true)
	e.state.Pause()
	_ = e.exporter.ForceFlush()
}

func (e *Engine) Resume() {
	e.state.Resume()
	e.paused.Store(false)
}

func (e *Engine) Stop() {
	e.stopOnce.Do(func() { close(e.stopCh) })
}

func (e *Engine) Run(ctx context.Context, domains []string) error {
	e.dashboard.Reset()
	e.pageBufMu.Lock()
	e.pageBuf = nil
	e.pageBufMu.Unlock()

	var todo []string
	for _, d := range domains {
		if u, err := url.Parse(d); err == nil && u.Host != "" {
			host := NormalizeHost(u.Host)
			if e.state.IsFinished(host) {
				e.log("[Memory] Skip " + host + " (déjà crawlé)")
				continue
			}
		}
		todo = append(todo, d)
	}
	if len(todo) == 0 {
		e.log("[Memory] Tous les domaines déjà traités — génération dorks uniquement")
		e.finalize(domains, false)
		return nil
	}

	e.setPhase(1, "Phase 1/4 — Crawling")

	monitorDone := make(chan struct{})
	go e.monitorLoop(monitorDone)

	stopped := atomic.Bool{}

	domainCh := make(chan string)
	var wg sync.WaitGroup
	workers := e.cfg.Workers
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range domainCh {
				if !e.acquireWorkerSlot(ctx) {
					return
				}
				select {
				case <-ctx.Done():
					e.active.Add(-1)
					return
				case <-e.stopCh:
					e.active.Add(-1)
					stopped.Store(true)
					return
				default:
					e.crawlDomain(ctx, d)
					e.active.Add(-1)
				}
			}
		}()
	}

	for _, d := range todo {
		e.dashboard.Ensure(d)
		select {
		case <-ctx.Done():
			close(domainCh)
			e.waitWorkers(&wg)
			close(monitorDone)
			e.runPostCrawlPhases()
			e.finalize(domains, false)
			return ctx.Err()
		case <-e.stopCh:
			close(domainCh)
			e.waitWorkers(&wg)
			close(monitorDone)
			stopped.Store(true)
			e.runPostCrawlPhases()
			e.finalize(domains, true)
			return nil
		default:
			domainCh <- d
		}
	}
	close(domainCh)
	e.waitWorkers(&wg)
	close(monitorDone)

	e.runPostCrawlPhases()
	e.finalize(domains, stopped.Load())
	return nil
}

func (e *Engine) acquireWorkerSlot(ctx context.Context) bool {
	for {
		snap := e.throttle.Current()
		if e.active.Load() < int32(snap.Workers) {
			e.active.Add(1)
			return true
		}
		if !e.waitBrief(ctx) {
			return false
		}
	}
}

func (e *Engine) waitBrief(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-e.stopCh:
		return false
	default:
		if e.paused.Load() {
			_ = e.exporter.ForceFlush()
		}
		for e.paused.Load() {
			select {
			case <-ctx.Done():
				return false
			case <-e.stopCh:
				return false
			default:
				time.Sleep(200 * time.Millisecond)
			}
		}
		time.Sleep(100 * time.Millisecond)
		return true
	}
}

func (e *Engine) waitWorkers(wg *sync.WaitGroup) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(throttle.GracefulShutdown):
		e.log("[Stop] Timeout gracieux 30s — arrêt forcé")
	}
	_ = e.exporter.ForceFlush()
}

func (e *Engine) bufferPage(host, rawURL, body string, links []string) {
	e.pageBufMu.Lock()
	defer e.pageBufMu.Unlock()
	e.pageBuf = append(e.pageBuf, pageRecord{
		host:  host,
		url:   rawURL,
		body:  body,
		links: append([]string(nil), links...),
	})
}

func (e *Engine) bufferedPageCount() int {
	e.pageBufMu.Lock()
	defer e.pageBufMu.Unlock()
	return len(e.pageBuf)
}

func (e *Engine) runPostCrawlPhases() {
	if e.bufferedPageCount() == 0 {
		return
	}
	e.setPhase(2, "Phase 2/4 — Keyword extraction")
	e.runKeywordPhase()
	e.setPhase(3, "Phase 3/4 — SQLi parameter scoring")
	e.runParamPhase()
}

func (e *Engine) runKeywordPhase() {
	e.pageBufMu.Lock()
	pages := append([]pageRecord(nil), e.pageBuf...)
	e.pageBufMu.Unlock()

	for _, rec := range pages {
		var doc *html.Node
		if rec.body != "" {
			doc, _ = html.Parse(strings.NewReader(rec.body))
		}
		kwResults := e.kw.ExtractPage(rec.host, rec.url, doc)
		if len(kwResults) > 0 {
			e.dashboard.AddKeywords(len(kwResults))
		}
	}
	e.emitSnapshot(phaseLabel(2), true, "")
}

func (e *Engine) runParamPhase() {
	e.pageBufMu.Lock()
	pages := append([]pageRecord(nil), e.pageBuf...)
	e.pageBufMu.Unlock()

	for _, rec := range pages {
		for _, link := range rec.links {
			link = canonicalURL(link)
			e.exportScoredParams(rec.host, e.scorer.ScoreURL(rec.host, link, e.cfg.MinParamScore))
		}
		e.exportScoredParams(rec.host, e.scorer.ScoreURL(rec.host, rec.url, e.cfg.MinParamScore))
	}
	e.emitSnapshot(phaseLabel(3), true, "")
}

func (e *Engine) finalize(domains []string, wasStopped bool) {
	e.setPhase(4, "Phase 4/4 — Auto-assemble dorks")
	preview := e.generateDorks(domains)
	if wasStopped {
		e.log("[Phase 4] Export partiel après arrêt")
	}
	e.emitSnapshot("Complete", false, preview)
	dorksPath := e.exporter.DorksPath()
	if err := e.exporter.Close(); err != nil {
		e.log("[Export] Erreur fermeture: " + err.Error())
		return
	}
	if e.events.OnDorksDone != nil && dorksPath != "" {
		e.events.OnDorksDone(dorksPath)
	}
}

func (e *Engine) monitorLoop(done <-chan struct{}) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			snap := e.throttle.Refresh(countGoroutines())
			acc, rej := e.scorer.Stats()
			if e.events.OnSnapshot != nil {
				e.emitSnapshot(phaseLabel(int(e.phase.Load())), true, "")
			} else {
				e.dashboard.Render(os.Stdout, snap, e.scorer.Decisions(), acc, rej)
			}
		}
	}
}

func (e *Engine) crawlDomain(ctx context.Context, seed string) {
	domainCtx, cancel := context.WithTimeout(ctx, e.cfg.TimeoutPerDom)
	defer cancel()

	u, err := url.Parse(seed)
	if err != nil {
		return
	}
	hostKey := NormalizeHost(u.Host)
	e.log("[Phase 1] Crawling " + hostKey)
	progress := e.state.Get(hostKey)
	if progress.Finished {
		return
	}

	queue := make([]string, 0, len(progress.Queue)+1)
	visited := map[string]bool{}
	queued := map[string]bool{}
	for _, v := range progress.Visited {
		c := canonicalURL(v)
		visited[c] = true
		queued[c] = true
	}
	for _, raw := range progress.Queue {
		c := canonicalURL(raw)
		if visited[c] || queued[c] {
			continue
		}
		queued[c] = true
		queue = append(queue, c)
	}
	if len(queue) == 0 && !progress.Finished {
		seedURL := canonicalURL(seed)
		if !visited[seedURL] && !queued[seedURL] {
			queued[seedURL] = true
			queue = append(queue, seedURL)
		}
	}

	pages := progress.Pages
	errors := progress.Errors

	for len(queue) > 0 {
		select {
		case <-domainCtx.Done():
			e.persist(hostKey, pages, errors, false, visited, queue)
			_ = e.exporter.ForceFlush()
			return
		case <-e.stopCh:
			e.persist(hostKey, pages, errors, false, visited, queue)
			_ = e.exporter.ForceFlush()
			return
		default:
			if !e.waitBrief(domainCtx) {
				e.persist(hostKey, pages, errors, false, visited, queue)
				_ = e.exporter.ForceFlush()
				return
			}
		}

		snap := e.throttle.Current()
		if pages >= snap.PageLimit {
			break
		}

		rawURL := queue[0]
		queue = queue[1:]
		delete(queued, rawURL)
		if visited[rawURL] {
			continue
		}
		visited[rawURL] = true

		depth := urlDepth(seed, rawURL)
		if depth > snap.Depth {
			continue
		}

		time.Sleep(time.Duration(snap.DelayMS) * time.Millisecond)

		body, links, err := e.fetch(domainCtx, rawURL)
		if err != nil {
			errors++
			e.dashboard.UpdateDomain(hostKey, pages, errors, false)
			continue
		}
		pages++

		e.bufferPage(hostKey, rawURL, body, links)
		for _, link := range links {
			link = canonicalURL(link)
			if sameHost(u, link) && !visited[link] && !queued[link] {
				queued[link] = true
				queue = append(queue, link)
			}
		}

		e.dashboard.UpdateDomain(hostKey, pages, errors, false)
		e.persist(hostKey, pages, errors, false, visited, queue)
	}

	e.dashboard.UpdateDomain(hostKey, pages, errors, true)
	e.persist(hostKey, pages, errors, true, visited, queue)
	_ = e.exporter.ForceFlush()
}

func (e *Engine) exportScoredParams(domain string, scored []params.Result) {
	if len(scored) == 0 {
		return
	}
	e.dashboard.AddParams(len(scored))
}

func (e *Engine) fetch(ctx context.Context, rawURL string) (string, []string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", e.cfg.UserAgent)

	resp, err := e.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	const maxBody = 2 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return "", nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return string(body), nil, nil
	}
	links := extractLinks(rawURL, doc)
	return string(body), links, nil
}

func extractLinks(base string, doc *html.Node) []string {
	baseURL, _ := url.Parse(base)
	var links []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					if abs := resolve(baseURL, attr.Val); abs != "" {
						links = append(links, abs)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return links
}

func resolve(base *url.URL, href string) string {
	u, err := base.Parse(href)
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	u.Fragment = ""
	s := canonicalURL(u.String())
	return s
}

func sameHost(seed *url.URL, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Host == seed.Host
}

func urlDepth(seed, target string) int {
	su, _ := url.Parse(seed)
	tu, _ := url.Parse(target)
	if su == nil || tu == nil {
		return 0
	}
	sp := strings.Trim(su.Path, "/")
	tp := strings.Trim(tu.Path, "/")
	if tp == "" {
		return 0
	}
	if sp == "" {
		return strings.Count(tp, "/") + 1
	}
	if strings.HasPrefix(tp, sp) {
		rel := strings.TrimPrefix(tp, sp)
		rel = strings.Trim(rel, "/")
		if rel == "" {
			return 0
		}
		return strings.Count(rel, "/") + 1
	}
	return strings.Count(tp, "/") + 1
}

func (e *Engine) persist(host string, pages, errors int, finished bool, visited map[string]bool, queue []string) {
	vis := make([]string, 0, len(visited))
	for u := range visited {
		vis = append(vis, u)
	}
	_ = e.state.Update(host, func(p *state.DomainProgress) {
		p.Pages = pages
		p.Errors = errors
		p.Finished = finished
		p.Visited = vis
		p.Queue = append([]string{}, queue...)
	})
}

func phaseLabel(n int) string {
	switch n {
	case 1:
		return "Phase 1/4 — Crawling"
	case 2:
		return "Phase 2/4 — Keyword extraction"
	case 3:
		return "Phase 3/4 — SQLi parameter scoring"
	case 4:
		return "Phase 4/4 — Auto-assemble dorks"
	default:
		return "Idle"
	}
}

func countGoroutines() int {
	return runtime.NumGoroutine()
}
