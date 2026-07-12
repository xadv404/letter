package crawler

import (
	"fmt"

	"github.com/xadv404/letter/internal/dorks"
)

func (e *Engine) generateDorks(domains []string) {
	for _, domain := range domains {
		host, tld := dorks.SiteScope(domain)

		topKW := e.kw.TopForDomain(host, 30)
		keywords := make([]string, 0, len(topKW))
		for _, r := range topKW {
			keywords = append(keywords, r.Keyword)
		}

		topParams := e.scorer.TopForDorks(host, 20)
		parameters := make([]string, 0, len(topParams))
		for _, pr := range topParams {
			parameters = append(parameters, pr.Name)
		}

		if len(keywords) == 0 {
			keywords = []string{"admin", "user", "search"}
		}
		if len(parameters) == 0 {
			parameters = []string{"id", "search_term", "filter_by"}
		}

		opts := dorks.Options{
			Host:       host,
			TLD:        tld,
			Keywords:   keywords,
			Parameters: parameters,
		}

		fmt.Printf("[Phase 4] Dorks for %s — %d keywords × %d params (%d templates/pair)\n",
			host, len(keywords), len(parameters), dorks.TemplateCount())

		fmt.Print(dorks.Preview(opts, 5))

		for _, dork := range e.dorks.Generate(opts) {
			_ = e.exporter.WriteDork(dork)
		}
	}
}
