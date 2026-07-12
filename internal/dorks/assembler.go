package dorks

import (
	"fmt"
	"strings"
)

// Assembler limits — mimics what a skilled human would combine (not blind cartesian).
const (
	MaxAssembledDorks = 600
	MaxKWClone        = 18
	MaxParamsAssemble = 20
	MaxPathsAssemble  = 8
	MaxPhrasesClone   = 8
)

// AssembledDork is a ready-to-run Google query with quality rating.
type AssembledDork struct {
	Dork    string
	TypeID  string
	Family  string
	Volume  string
	Param   string
	Keyword string
	Path    string
	Score   int
	Tier    string
}

// Assemble builds final dorks from materials (types × keywords × params).
func Assemble(m Materials) []AssembledDork {
	params := m.Params
	if len(params) == 0 {
		params = []string{"id", "cat", "pid", "product_id", "item_id"}
	}
	if len(params) > MaxParamsAssemble {
		params = params[:MaxParamsAssemble]
	}
	paths := m.Paths
	if len(paths) > MaxPathsAssemble {
		paths = paths[:MaxPathsAssemble]
	}
	keywords := m.Keywords
	if len(keywords) > MaxKWClone {
		keywords = keywords[:MaxKWClone]
	}
	phrases := m.Phrases
	if len(phrases) > MaxPhrasesClone {
		phrases = phrases[:MaxPhrasesClone]
	}

	seen := map[string]struct{}{}
	var out []AssembledDork
	add := func(dork, typeID, family, volume, param, kw, path string) {
		dork = strings.TrimSpace(dork)
		if dork == "" || strings.Contains(dork, "{") {
			return
		}
		if _, ok := seen[dork]; ok {
			return
		}
		if len(out) >= MaxAssembledDorks {
			return
		}
		seen[dork] = struct{}{}
		out = append(out, AssembledDork{
			Dork: dork, TypeID: typeID, Family: family, Volume: volume,
			Param: param, Keyword: kw, Path: path,
		})
	}

	typesByFamily := map[string][]DorkType{}
	for _, t := range m.Types {
		typesByFamily[t.Family] = append(typesByFamily[t.Family], t)
	}

	// 1) Volume + error — every injectable param (max URLs).
	for _, t := range typesByFamily["param_volume"] {
		for _, pm := range params {
			add(applySlots(t.Pattern, slotMap(pm, "", "", "")), t.ID, t.Family, t.Volume, pm, "", "")
		}
	}
	for _, t := range typesByFamily["sqli_error"] {
		for _, pm := range params {
			add(applySlots(t.Pattern, slotMap(pm, "", "", "")), t.ID, t.Family, t.Volume, pm, "", "")
		}
	}

	// 2) Path × param (crawled layout).
	for _, t := range typesByFamily["path_context"] {
		for _, path := range paths {
			for _, pm := range params {
				if strings.Contains(t.Pattern, "{kw}") {
					continue
				}
				add(applySlots(t.Pattern, slotMap(pm, "", path, "")), t.ID, t.Family, t.Volume, pm, "", path)
			}
		}
	}

	// 3) Clone — keyword × param (find similar sites).
	for _, t := range typesByFamily["keyword_match"] {
		for _, kw := range keywords {
			for _, pm := range params {
				add(applySlots(t.Pattern, slotMap(pm, kw, "", "")), t.ID, t.Family, t.Volume, pm, kw, "")
			}
		}
		for _, ph := range phrases {
			for _, pm := range params {
				add(applySlots(t.Pattern, slotMap(pm, ph, "", "")), t.ID, t.Family, t.Volume, pm, ph, "")
			}
		}
	}

	// 4) Path × keyword × param (thematic clones in seed directories).
	for _, t := range typesByFamily["path_context"] {
		if !strings.Contains(t.Pattern, "{kw}") {
			continue
		}
		for _, path := range paths {
			for _, kw := range keywords[:cloneKWLimit(len(keywords))] {
				for _, pm := range params[:cloneParamLimit(len(params))] {
					add(applySlots(t.Pattern, slotMap(pm, kw, path, "")), t.ID, t.Family, t.Volume, pm, kw, path)
				}
			}
		}
	}

	// 5) Multi-param pairs from crawl.
	for _, t := range typesByFamily["multi_param"] {
		for i := 0; i < len(params) && i < 8; i++ {
			for j := i + 1; j < len(params) && j < i+4; j++ {
				slots := slotMap(params[i], "", "", "")
				slots["param2"] = params[j]
				add(applySlots(t.Pattern, slots), t.ID, t.Family, t.Volume, params[i], "", "")
			}
		}
	}

	return out
}

// AssembleStrings returns dork lines only.
func AssembleStrings(m Materials) []string {
	assembled := Assemble(m)
	out := make([]string, len(assembled))
	for i, a := range assembled {
		out[i] = a.Dork
	}
	return out
}

func slotMap(param, kw, path, param2 string) map[string]string {
	return map[string]string{
		SlotParam: param,
		SlotKW:    kw,
		SlotPath:  path,
		"param2":  param2,
	}
}

func applySlots(pattern string, slots map[string]string) string {
	out := pattern
	for k, v := range slots {
		if v == "" {
			continue
		}
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

func cloneKWLimit(n int) int {
	if n > 10 {
		return 10
	}
	return n
}

func cloneParamLimit(n int) int {
	if n > 6 {
		return 6
	}
	return n
}

// PreviewAssembled summarizes auto-generated dorks with scores.
func PreviewAssembled(m Materials, limit int) string {
	all := RankAssembled(m)
	elite, high := 0, 0
	for _, d := range all {
		switch d.Tier {
		case TierElite:
			elite++
		case TierHigh:
			high++
		}
	}
	show := all
	if limit > 0 && len(show) > limit {
		show = show[:limit]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Assembled %d dorks — ELITE: %d | HIGH: %d (scored & sorted)\n", len(all), elite, high)
	for _, d := range show {
		b.WriteString(fmt.Sprintf("  [%d %s] %s\n", d.Score, d.Tier, d.Dork))
	}
	return b.String()
}
