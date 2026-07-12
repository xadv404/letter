package crawler

import (
	"fmt"
	"strings"

	"github.com/xadv404/letter/internal/dorks"
)

func (e *Engine) generateDorks(domains []string) string {
	fp := dorks.NewFingerprint()

	for _, domain := range domains {
		host, _ := dorks.SiteScope(domain)

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

	e.log(fmt.Sprintf("[Phase 4] %d dorks pour trouver des clones similaires vulnérables SQLi", len(generated)))
	preview := dorks.PreviewList(generated, 12)
	e.log(preview)
	return strings.Join([]string{
		fmt.Sprintf("Theme fingerprint: %d kw / %d phrases / %d params / %d paths",
			len(fp.Keywords), len(fp.Phrases), len(fp.Parameters), len(fp.Paths)),
		fmt.Sprintf("Generated %d dorks", len(generated)),
		preview,
	}, "\n")
}
