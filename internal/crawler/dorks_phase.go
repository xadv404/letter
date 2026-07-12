package crawler

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xadv404/letter/internal/dorks"
	"github.com/xadv404/letter/internal/keywords"
)

func (e *Engine) generateDorks(domains []string) string {
	fp := dorks.NewFingerprint()
	var provenURLs []string

	for _, domain := range domains {
		rawHost, _ := dorks.SiteScope(domain)
		host := NormalizeHost(rawHost)

		for _, r := range e.kw.TopForDomain(host, 40) {
			fp.AddTerm(r.Keyword)
		}
		for _, path := range e.kw.TopPathsForDomain(host, 50) {
			fp.AddPath(path)
		}
		for _, r := range e.scorer.TopInjectableParams(host, 100) {
			fp.AddParameter(r.Name)
		}
		for _, flagged := range e.scorer.FlaggedURLs(host) {
			provenURLs = append(provenURLs, flagged.URL)
			fp.AddURLPaths(flagged.URL)
			for _, p := range flagged.HighParams {
				fp.AddParameter(p)
			}
			if u, err := url.Parse(flagged.URL); err == nil {
				for name := range u.Query() {
					fp.AddParameter(name)
				}
			}
		}
	}

	rawParams := append([]string{}, fp.Parameters...)
	for _, p := range dorks.ExpandInjectableParams(rawParams) {
		fp.AddParameter(p)
	}

	seedTerms := append([]string{}, fp.Keywords...)
	seedTerms = append(seedTerms, fp.Phrases...)
	expanded := keywords.ExpandAutocomplete(seedTerms, 8)
	for _, term := range expanded {
		fp.AddTerm(term)
	}
	if len(expanded) > 0 {
		e.log(fmt.Sprintf("[Enrichment] +%d keywords via autocomplete", len(expanded)))
	}

	fp.Finalize()

	if !fp.Viable() {
		msg := "[Phase 4] Aucun param injectable — crawl plus de pages avec query strings"
		e.log(msg)
		return "No dorks generated."
	}

	e.log(fmt.Sprintf(
		"[Phase 4] Empreinte (%d seeds): %d params, %d paths, %d URLs prouvées",
		len(domains), len(fp.Parameters), len(fp.Paths), len(provenURLs),
	))

	set := e.dorks.GenerateExploitable(*fp, provenURLs)

	for _, dork := range set.Exploitable {
		_ = e.exporter.WriteExploitableDork(dork)
	}
	for _, dork := range set.All {
		_ = e.exporter.WriteDork(dork)
	}

	e.log(fmt.Sprintf(
		"[Phase 4] %d dorks exploitables (PRIORITAIRES) + %d volume → dorks_exploitable.txt d'abord",
		len(set.Exploitable), len(set.All),
	))
	preview := dorks.PreviewList(set.Exploitable, 12)
	e.log(preview)
	return strings.Join([]string{
		fmt.Sprintf("Exploitable: %d dorks (run first) | Volume: %d dorks",
			len(set.Exploitable), len(set.All)),
		fmt.Sprintf("Files: dorks_exploitable.txt → dorks.txt"),
		preview,
	}, "\n")
}
