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

	pauseCh  chan struct{}
	resumeCh chan struct{}
	stopCh   chan struct{}
	stopOnce sync.Once

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
	exp, err := export.New(cfg.OutputDir)
	if err != nil {
		return nil, err
	}
	return &Engine{
		cfg:            cfg,
		throttle:       throttle.New(cfg.DelayMS, cfg.Depth, cfg.PageLimit, cfg.Workers),
		exporter:       exp,
		state:          st,
		dashboard:      monitor.New(),
		kw:             keywords.New(config.MaxKeywords),
		scorer:         params.New(config.MaxParams, cfg.OutputDir),
		dorks:          dorks.New(),
		client:         &http.Client{Timeout: 15 * time.Second},
		pauseCh:        make(chan struct{}, 1),
		resumeCh:       make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
		events:         events,
	}, nil
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
	ui := e.dashboard.Snapshot(
		int(e.phase.Load()),
		phaseLabel,
		snap,
		e.scorer.Decisions(),
		acc, rej,
		dorkPreview,
		running,
	)
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

func (e *Engine) Pause()  { e.state.Pause(); e.pauseCh <- struct{}{} }
func (e *Engine) Resume() { e.state.Resume(); e.resumeCh <- struct{}{} }
func (e *Engine) Stop() {
	e.stopOnce.Do(func() { close(e.stopCh) })
}

func (e *Engine) Run(ctx context.Context, domains []string) error {
	e.dashboard.Reset()
	e.setPhase(1, "Phase 1/4 — Crawling")

	monitorDone := make(chan struct{})
	go e.monitorLoop(monitorDone)

	for _, d := range domains {
		e.dashboard.Ensure(d)
		select {
		case <-ctx.Done():
			close(monitorDone)
			return ctx.Err()
		case <-e.stopCh:
			close(monitorDone)
			return nil
		default:
			e.crawlDomain(ctx, d)
		}
	}

	close(monitorDone)

	e.setPhase(4, "Phase 4/4 — Google Dork generation")
	preview := e.generateDorks(domains)
	e.emitSnapshot("Complete", false, preview)
	dorksPath := e.exporter.DorksPath()
	if err := e.exporter.Close(); err != nil {
		return err
	}
	if e.events.OnDorksDone != nil && dorksPath != "" {
		e.events.OnDorksDone(dorksPath)
	}
	return nil
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
	hostKey := u.Host
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
			return
		case <-e.stopCh:
			e.persist(hostKey, pages, errors, false, visited, queue)
			return
		case <-e.pauseCh:
			e.persist(hostKey, pages, errors, false, visited, queue)
			<-e.resumeCh
		default:
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

		e.setPhase(2, "Phase 2/4 — Keyword extraction")

		doc, err := html.Parse(strings.NewReader(body))
		if err == nil {
			if krs := e.kw.Extract(hostKey, doc); len(krs) > 0 {
				e.dashboard.AddKeywords(len(krs))
			}
		}

		for _, link := range links {
			link = canonicalURL(link)
			if sameHost(u, link) && !visited[link] && !queued[link] {
				queued[link] = true
				queue = append(queue, link)
			}
			if scored := e.scorer.ScoreURL(hostKey, link, e.cfg.MinParamScore); len(scored) > 0 {
				e.dashboard.AddParams(len(scored))
			}
		}

		e.setPhase(3, "Phase 3/4 — SQLi parameter scoring")
		if scored := e.scorer.ScoreURL(hostKey, rawURL, e.cfg.MinParamScore); len(scored) > 0 {
			e.dashboard.AddParams(len(scored))
		}

		e.dashboard.UpdateDomain(hostKey, pages, errors, false)
		e.persist(hostKey, pages, errors, false, visited, queue)
	}

	e.dashboard.UpdateDomain(hostKey, pages, errors, true)
	e.persist(hostKey, pages, errors, true, visited, queue)
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
		return "Phase 4/4 — Google Dork generation"
	default:
		return "Idle"
	}
}

func countGoroutines() int {
	return runtime.NumGoroutine()
}
