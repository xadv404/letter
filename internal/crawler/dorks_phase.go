package crawler

import (
	"fmt"
	"strings"

	"github.com/xadv404/letter/internal/dorks"
	"github.com/xadv404/letter/internal/keywords"
)

func (e *Engine) generateDorks(domains []string) string {
	fp := dorks.NewFingerprint()

	for _, domain := range domains {
		rawHost, _ := dorks.SiteScope(domain)
		host := NormalizeHost(rawHost)

		for _, r := range e.kw.TopForDomain(host, 50) {
			fp.AddTerm(r.Keyword)
		}
		for _, path := range e.kw.TopPathsForDomain(host, 30) {
			fp.AddPath(path)
		}
		for _, r := range e.scorer.TopForDorks(host, 45) {
			fp.AddParameter(r.Name)
		}
		for _, flagged := range e.scorer.FlaggedURLs(host) {
			fp.AddURLPaths(flagged.URL)
			for _, p := range flagged.HighParams {
				fp.AddParameter(p)
			}
		}
	}

	seedTerms := append([]string{}, fp.Keywords...)
	seedTerms = append(seedTerms, fp.Phrases...)
	expanded := keywords.ExpandAutocomplete(seedTerms, 6)
	for _, term := range expanded {
		fp.AddTerm(term)
	}
	if len(expanded) > 0 {
		e.log(fmt.Sprintf("[Enrichment] +%d keywords via autocomplete", len(expanded)))
	}

	fp.Finalize()

	if !fp.Viable() {
		msg := "[Phase 4] Données insuffisantes — crawl plus de pages sur les seeds"
		e.log(msg)
		return "No dorks generated."
	}

	e.log(fmt.Sprintf(
		"[Phase 4] Empreinte thème (%d seeds): %d keywords, %d phrases, %d params, %d paths",
		len(domains), len(fp.Keywords), len(fp.Phrases), len(fp.Parameters), len(fp.Paths),
	))

	generated := e.dorks.GenerateFingerprint(*fp)
	for _, dork := range generated {
		_ = e.exporter.WriteDork(dork)
	}

	e.log(fmt.Sprintf("[Phase 4] %d dorks (%d templates) pour clones SQLi", len(generated), dorks.TemplateCount()))
	preview := dorks.PreviewList(generated, 12)
	e.log(preview)
	return strings.Join([]string{
		fmt.Sprintf("Theme fingerprint: %d kw / %d phrases / %d params / %d paths",
			len(fp.Keywords), len(fp.Phrases), len(fp.Parameters), len(fp.Paths)),
		fmt.Sprintf("Generated %d dorks", len(generated)),
		preview,
	}, "\n")
}
