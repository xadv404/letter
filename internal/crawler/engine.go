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
	stopCh chan struct{}
}

func New(cfg config.CrawlConfig) (*Engine, error) {
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
		stopCh: make(chan struct{}),
	}, nil
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
func (e *Engine) Stop()   { close(e.stopCh) }

func (e *Engine) Run(ctx context.Context, domains []string) error {
	domainCh := make(chan string)
	var wg sync.WaitGroup
	workerCount := e.cfg.Workers

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for domain := range domainCh {
				e.crawlDomain(ctx, domain)
			}
		}()
	}

	monitorDone := make(chan struct{})
	go e.monitorLoop(monitorDone)

	for _, d := range domains {
		e.dashboard.Ensure(d)
		domainCh <- d
	}
	close(domainCh)
	wg.Wait()

	close(monitorDone)
	e.generateDorks(domains)
	return e.exporter.Close()
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
			e.dashboard.Render(os.Stdout, snap, e.scorer.Decisions(), acc, rej)
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
	progress := e.state.Get(hostKey)
	if progress.Finished {
		return
	}

	queue := append([]string{}, progress.Queue...)
	visited := map[string]bool{}
	for _, v := range progress.Visited {
		visited[v] = true
	}
	if len(queue) == 0 {
		queue = append(queue, seed)
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

		doc, err := html.Parse(strings.NewReader(body))
		if err == nil {
			for _, kr := range e.kw.Extract(hostKey, doc) {
				_ = e.exporter.WriteKeyword(kr.Domain, kr.Keyword, kr.Weight, kr.Source)
				e.dashboard.AddKeywords(1)
			}
		}

		for _, link := range links {
			if sameHost(u, link) && !visited[link] {
				queue = append(queue, link)
			}
			for _, pr := range e.scorer.ScoreURL(hostKey, link, e.cfg.MinParamScore) {
				_ = e.exporter.WriteParameter(pr.Domain, pr.URL, pr.Name, pr.Score, string(pr.Tier), pr.Matched)
				e.dashboard.AddParams(1)
			}
		}

		for _, pr := range e.scorer.ScoreURL(hostKey, rawURL, e.cfg.MinParamScore) {
			_ = e.exporter.WriteParameter(pr.Domain, pr.URL, pr.Name, pr.Score, string(pr.Tier), pr.Matched)
			e.dashboard.AddParams(1)
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
	return u.String()
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

func (e *Engine) generateDorks(domains []string) {
	for _, domain := range domains {
		u, err := url.Parse(domain)
		if err != nil {
			continue
		}
		host := u.Host
		top := e.kw.TopForDomain(host, 50)
		kws := make([]string, 0, len(top))
		for _, r := range top {
			kws = append(kws, r.Keyword)
		}
		prms := make([]string, 0, 50)
		for _, pr := range e.scorer.TopForDomain(host, 50) {
			prms = append(prms, pr.Name)
		}
		if len(kws) == 0 {
			kws = []string{"admin", "login"}
		}
		if len(prms) == 0 {
			prms = []string{"id", "search", "page"}
		}
		preview := dorks.Preview(host, kws, prms, 5)
		fmt.Println(preview)
		for _, dork := range e.dorks.Generate(host, kws, prms, 0) {
			_ = e.exporter.WriteDork(dork)
		}
	}
}

func countGoroutines() int {
	return runtime.NumGoroutine()
}
