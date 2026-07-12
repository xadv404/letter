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
	var kwList, phList []string

	for _, domain := range domains {
		rawHost, _ := dorks.SiteScope(domain)
		host := NormalizeHost(rawHost)

		for _, r := range e.kw.TopForDomain(host, 40) {
			fp.AddTerm(r.Keyword)
			kwList = append(kwList, r.Keyword)
		}
		for _, path := range e.kw.TopPathsForDomain(host, 30) {
			fp.AddPath(path)
		}
		for _, r := range e.scorer.TopInjectableParams(host, 40) {
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

	for _, r := range e.kw.Top(50) {
		fp.AddTerm(r.Keyword)
		kwList = append(kwList, r.Keyword)
	}

	seedTerms := uniqueStrings(kwList)
	expanded := keywords.ExpandAutocomplete(seedTerms, 15)
	for _, term := range expanded {
		fp.AddTerm(term)
		kwList = append(kwList, term)
	}
	if len(expanded) > 0 {
		e.log(fmt.Sprintf("[Keywords] +%d termes via autocomplete", len(expanded)))
	}

	for _, r := range e.kw.Top(60) {
		term := r.Keyword
		if strings.Contains(term, " ") {
			phList = append(phList, term)
		} else {
			kwList = append(kwList, term)
		}
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

	if err := e.exporter.WriteMaterials(materials); err != nil {
		e.log("[Phase 4] Erreur export: " + err.Error())
		return "Export failed."
	}

	e.log(fmt.Sprintf(
		"[Phase 4] %d dorks notés | ELITE+HIGH: %d | %d types | %d keywords | %d params",
		len(assembled), countEliteHigh(assembled), len(materials.Types), len(materials.Keywords)+len(materials.Phrases), len(materials.Params),
	))
	preview := dorks.PreviewAssembled(materials, 12)
	e.log(preview)
	return strings.Join([]string{
		fmt.Sprintf("dorks.txt: %d requêtes notées (score | tier | family | dork)", len(assembled)),
		fmt.Sprintf("dorktypes.txt: %d | keywords.txt: %d | params.txt: %d",
			len(materials.Types), len(materials.Keywords)+len(materials.Phrases), len(materials.Params)),
		preview,
	}, "\n")
}

func countEliteHigh(ranked []dorks.AssembledDork) int {
	n := 0
	for _, d := range ranked {
		if d.Tier == dorks.TierElite || d.Tier == dorks.TierHigh {
			n++
		}
	}
	return n
}

func (e *Engine) buildParamScores(domains []string) map[string]int {
	out := map[string]int{}
	for _, domain := range domains {
		rawHost, _ := dorks.SiteScope(domain)
		host := NormalizeHost(rawHost)
		for _, r := range e.scorer.TopInjectableParams(host, 80) {
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
	for _, r := range e.kw.Top(120) {
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
