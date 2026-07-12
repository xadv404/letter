package crawler

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xadv404/letter/internal/dorks"
)

func (e *Engine) generateDorks(domains []string) string {
	fp := dorks.NewFingerprint()

	for _, domain := range domains {
		rawHost, _ := dorks.SiteScope(domain)
		host := NormalizeHost(rawHost)

		for _, r := range e.scorer.TopInjectableParams(host, 50) {
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

	for _, p := range dorks.ExpandInjectableParams(fp.Parameters) {
		fp.AddParameter(p)
	}

	fp.Finalize()

	if !fp.Viable() {
		e.log("[Phase 4] Aucun param — crawl des seeds avec query strings (?id=, ?cat=)")
		return "No dorks generated."
	}

	set := dorks.GenerateURLVolume(fp.Parameters)

	for _, dork := range set.All {
		_ = e.exporter.WriteDork(dork)
	}

	e.log(fmt.Sprintf(
		"[Phase 4] %d dorks URL-VOLUME (cible 200-500k URLs) → dorks.txt",
		len(set.All),
	))
	preview := dorks.PreviewList(set.All, len(set.All))
	e.log(preview)
	return strings.Join([]string{
		fmt.Sprintf("%d dorks ultra-larges — ~5-10k URLs/dork → 200-500k total", len(set.All)),
		"Lance chaque dork sur Google (paginer au max)",
		preview,
	}, "\n")
}
