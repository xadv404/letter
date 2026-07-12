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

	for _, domain := range domains {
		rawHost, _ := dorks.SiteScope(domain)
		host := NormalizeHost(rawHost)

		for _, r := range e.kw.TopForDomain(host, 60) {
			fp.AddTerm(r.Keyword)
		}
		for _, path := range e.kw.TopPathsForDomain(host, 50) {
			fp.AddPath(path)
		}
		for _, r := range e.scorer.TopInjectableParams(host, 100) {
			fp.AddParameter(r.Name)
		}
		for _, flagged := range e.scorer.FlaggedURLs(host) {
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
	expanded := keywords.ExpandAutocomplete(seedTerms, 12)
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
		"[Phase 4] Empreinte injectable (%d seeds): %d params, %d keywords, %d paths",
		len(domains), len(fp.Parameters), len(fp.Keywords), len(fp.Paths),
	))

	generated := e.dorks.GenerateFingerprint(*fp)
	for _, dork := range generated {
		_ = e.exporter.WriteDork(dork)
	}

	e.log(fmt.Sprintf("[Phase 4] %d dorks blast injectables (volume max)", len(generated)))
	preview := dorks.PreviewList(generated, 12)
	e.log(preview)
	return strings.Join([]string{
		fmt.Sprintf("Injectable fingerprint: %d params / %d kw / %d paths",
			len(fp.Parameters), len(fp.Keywords), len(fp.Paths)),
		fmt.Sprintf("Generated %d dorks", len(generated)),
		preview,
	}, "\n")
}
