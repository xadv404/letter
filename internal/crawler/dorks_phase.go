package crawler

import (
	"strings"

	"github.com/xadv404/letter/internal/dorks"
)

func (e *Engine) generateDorks(domains []string) string {
	var previews []string
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

		msg := "[Phase 4] Dorks (sites similaires) depuis " + host
		e.log(msg)
		preview := dorks.Preview(opts, 8)
		previews = append(previews, preview)
		e.log(preview)

		for _, dork := range e.dorks.Generate(opts) {
			_ = e.exporter.WriteDork(dork)
		}
	}
	return strings.Join(previews, "\n")
}
