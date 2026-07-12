package dorks

import (
	"sort"
	"strings"
)

// Dork tiers — human-readable quality bands.
const (
	TierElite  = "ELITE"
	TierHigh   = "HIGH"
	TierMedium = "MEDIUM"
	TierLow    = "LOW"
)

var familyBaseScore = map[string]int{
	"sqli_error":    82,
	"keyword_match": 76,
	"path_context":  72,
	"param_volume":  68,
	"multi_param":   64,
}

// RankAssembled scores and sorts dorks (best first).
func RankAssembled(m Materials) []AssembledDork {
	out := Assemble(m)
	for i := range out {
		out[i].Score, out[i].Tier = RateDork(out[i], m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Dork < out[j].Dork
		}
		return out[i].Score > out[j].Score
	})
	return out
}

// RateDork returns a 0–100 score and tier for one assembled dork.
func RateDork(d AssembledDork, m Materials) (int, string) {
	score := familyBaseScore[d.Family]
	if score == 0 {
		score = 60
	}

	if d.Volume == "high" {
		score += 6
	}

	score += paramBoost(d.Param, m.ParamScores)
	score += keywordBoost(d.Keyword, m.KeywordScores)

	switch {
	case d.Family == "sqli_error":
		score += 12
	case strings.Contains(d.Dork, "intext:") && strings.Contains(d.Dork, "inurl:"):
		score += 8
	case strings.Count(d.Dork, "inurl:") == 1 && !strings.Contains(d.Dork, "intext:"):
		score += 10
	}

	if strings.Contains(d.Dork, "filetype:") {
		score -= 4
	}
	if strings.Count(d.Dork, "inurl:") > 1 {
		score -= 6
	}
	if isRedundantScriptDork(d.Dork, d.Param) {
		score -= 12
	}
	if isNoiseParam(d.Param) {
		score -= 40
	}
	if d.Path != "" {
		score += 5
	}
	if IsExploitableParam(d.Param) {
		score += 8
	} else if !isPrimaryInjectable(d.Param) {
		score -= 10
	}

	score = clampScore(score)
	return score, scoreToTier(score)
}

func paramBoost(param string, scores map[string]int) int {
	if param == "" || len(scores) == 0 {
		if IsExploitableParam(param) {
			return 12
		}
		return 0
	}
	ps, ok := scores[strings.ToLower(param)]
	if !ok {
		if IsExploitableParam(param) {
			return 10
		}
		return 0
	}
	switch {
	case ps >= 85:
		return 22
	case ps >= 65:
		return 14
	case ps >= 50:
		return 6
	default:
		return 0
	}
}

func keywordBoost(kw string, scores map[string]int) int {
	if kw == "" {
		return 0
	}
	if len(scores) == 0 {
		if strings.Contains(kw, " ") {
			return 8
		}
		return 4
	}
	w, ok := scores[strings.ToLower(kw)]
	if !ok {
		return 3
	}
	if w >= 50 {
		return 15
	}
	if w >= 20 {
		return 10
	}
	return 5
}

func scoreToTier(score int) string {
	switch {
	case score >= 90:
		return TierElite
	case score >= 75:
		return TierHigh
	case score >= 60:
		return TierMedium
	default:
		return TierLow
	}
}

func clampScore(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// RankedStrings returns dork lines sorted by score.
func RankedStrings(m Materials) []string {
	ranked := RankAssembled(m)
	out := make([]string, len(ranked))
	for i, d := range ranked {
		out[i] = d.Dork
	}
	return out
}

// isRedundantScriptDork penalizes index.php?param= style duplicates.
func isRedundantScriptDork(dork, param string) bool {
	if param == "" {
		return false
	}
	for _, script := range []string{"index.php?", "item.php?", "list.php?", "news.php?", "forum.php?", "profile.php?", "product.php?", "view.php?"} {
		if strings.Contains(dork, "inurl:"+script+param+"=") {
			return true
		}
	}
	return false
}
