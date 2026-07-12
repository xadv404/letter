package crawler

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xadv404/letter/internal/config"
	"github.com/xadv404/letter/internal/dorks"
	"github.com/xadv404/letter/internal/keywords"
)

func (e *Engine) generateDorks(domains []string) string {
	fp := dorks.NewFingerprint()
	var kwList, phList []string

	rankedKW := e.kw.Top(config.MaxExportKeywords)

	for _, domain := range domains {
		rawHost, _ := dorks.SiteScope(domain)
		host := NormalizeHost(rawHost)

		for _, r := range e.kw.TopForDomain(host, 25) {
			fp.AddTerm(r.Keyword)
		}
		for _, path := range e.kw.TopPathsForDomain(host, 15) {
			fp.AddPath(path)
		}
		for _, r := range e.scorer.TopInjectableParams(host, 20) {
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

	for _, r := range rankedKW {
		term := r.Keyword
		fp.AddTerm(term)
		if strings.Contains(term, " ") {
			phList = append(phList, term)
		} else {
			kwList = append(kwList, term)
		}
	}

	seedTerms := uniqueStrings(kwList)
	expanded := keywords.ExpandAutocomplete(seedTerms, 10)
	for _, term := range expanded {
		fp.AddTerm(term)
		if !strings.Contains(term, " ") {
			kwList = append(kwList, term)
		}
	}
	if len(expanded) > 0 {
		e.log(fmt.Sprintf("[Keywords] +%d termes via autocomplete", len(expanded)))
	}

	fp.Finalize()
	kwList = uniqueStrings(kwList)
	phList = uniqueStrings(phList)

	if len(fp.Parameters) == 0 && len(kwList) == 0 {
		e.log("[Phase 4] Données insuffisantes — crawl plus de pages")
		return "No materials prepared."
	}

	materials := dorks.PrepareMaterials(*fp, kwList, phList)
	materials.ParamScores = e.buildParamScores(domains)
	materials.KeywordScores = e.buildKeywordScores()
	assembled := dorks.RankAssembled(materials)

	if err := e.exporter.WriteDorks(assembled); err != nil {
		e.log("[Phase 4] Erreur export: " + err.Error())
		return "Export failed."
	}

	e.log(fmt.Sprintf(
		"[Phase 4] %d dorks exportés (%d keywords utilisés en interne)",
		len(assembled), len(rankedKW),
	))
	preview := dorks.PreviewAssembled(materials, 12)
	e.log(preview)
	return strings.Join([]string{
		fmt.Sprintf("dorks.txt: %d requêtes notées", len(assembled)),
		preview,
	}, "\n")
}

func (e *Engine) buildParamScores(domains []string) map[string]int {
	out := map[string]int{}
	for _, domain := range domains {
		rawHost, _ := dorks.SiteScope(domain)
		host := NormalizeHost(rawHost)
		for _, r := range e.scorer.TopInjectableParams(host, 40) {
			name := strings.ToLower(r.Name)
			if prev, ok := out[name]; !ok || r.Score > prev {
				out[name] = r.Score
			}
		}
	}
	return out
}

func (e *Engine) buildKeywordScores() map[string]int {
	out := map[string]int{}
	for _, r := range e.kw.Top(config.MaxExportKeywords) {
		kw := strings.ToLower(r.Keyword)
		if prev, ok := out[kw]; !ok || r.Weight > prev {
			out[kw] = r.Weight
		}
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
