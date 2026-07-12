package dorks

import (
	"fmt"
	"strings"
)

// Assembler limits — human-like combinations, not blind cartesian.
const (
	MaxKWClone      = 12
	MaxPathsAssemble = 6
	MaxPhrasesClone = 6
	MaxCloneParams  = 8
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
	params := selectParamsForAssembly(m)
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
	typeIndex := typesByID(m.Types)
	for _, t := range m.Types {
		typesByFamily[t.Family] = append(typesByFamily[t.Family], t)
	}

	// 1) Volume — max 2 broad dorks per param (no script × param cartesian).
	for _, pm := range params {
		for _, t := range volumeTypesForParam(pm, paths, typeIndex) {
			add(applySlots(t.Pattern, slotMap(pm, "", "", "")), t.ID, t.Family, t.Volume, pm, "", "")
		}
	}

	// 2) SQLi errors — top params only, 1 template each.
	errorTypes := pickErrorTypes(typesByFamily["sqli_error"])
	errorParams := params
	if len(errorParams) > 8 {
		errorParams = errorParams[:8]
	}
	for _, pm := range errorParams {
		if len(errorTypes) == 0 {
			break
		}
		t := errorTypes[0]
		add(applySlots(t.Pattern, slotMap(pm, "", "", "")), t.ID, t.Family, t.Volume, pm, "", "")
	}

	// 3) Path × param — crawled paths only, one template per pair.
	for _, path := range paths {
		for _, pm := range params {
			if len(typesByFamily["path_context"]) == 0 {
				break
			}
			t := typesByFamily["path_context"][0]
			if strings.Contains(t.Pattern, "{kw}") {
				for _, candidate := range typesByFamily["path_context"] {
					if !strings.Contains(candidate.Pattern, "{kw}") {
						t = candidate
						break
					}
				}
			}
			if strings.Contains(t.Pattern, "{kw}") {
				continue
			}
			add(applySlots(t.Pattern, slotMap(pm, "", path, "")), t.ID, t.Family, t.Volume, pm, "", path)
		}
	}

	// 4) Clone — keyword × param, limited templates.
	kwTypes := pickKeywordTypes(typesByFamily["keyword_match"], MaxKeywordTemplatesPerKW)
	cloneParams := params
	if len(cloneParams) > MaxCloneParams {
		cloneParams = cloneParams[:MaxCloneParams]
	}
	for _, kw := range keywords {
		for _, pm := range cloneParams {
			for _, t := range kwTypes {
				add(applySlots(t.Pattern, slotMap(pm, kw, "", "")), t.ID, t.Family, t.Volume, pm, kw, "")
			}
		}
	}
	for _, ph := range phrases {
		phParams := cloneParams
		if len(phParams) > 4 {
			phParams = phParams[:4]
		}
		for _, pm := range phParams {
			if len(kwTypes) > 0 {
				t := kwTypes[0]
				add(applySlots(t.Pattern, slotMap(pm, ph, "", "")), t.ID, t.Family, t.Volume, pm, ph, "")
			}
		}
	}

	// 5) Path × keyword × param — top 3 kw × top 4 params, single template.
	pathKwType := findPathKeywordType(typesByFamily["path_context"])
	if pathKwType != nil {
		kwLim := cloneKWLimit(len(keywords))
		if kwLim > 3 {
			kwLim = 3
		}
		pmLim := cloneParamLimit(len(params))
		if pmLim > 4 {
			pmLim = 4
		}
		pathLim := len(paths)
		if pathLim > 3 {
			pathLim = 3
		}
		for _, path := range paths[:pathLim] {
			for _, kw := range keywords[:kwLim] {
				for _, pm := range params[:pmLim] {
					add(applySlots(pathKwType.Pattern, slotMap(pm, kw, path, "")), pathKwType.ID, pathKwType.Family, pathKwType.Volume, pm, kw, path)
				}
			}
		}
	}

	// 6) Multi-param — top 5 params, adjacent pairs only.
	for i := 0; i < len(params) && i < 5; i++ {
		for j := i + 1; j < len(params) && j < i+2; j++ {
			for _, t := range typesByFamily["multi_param"] {
				slots := slotMap(params[i], "", "", "")
				slots["param2"] = params[j]
				add(applySlots(t.Pattern, slots), t.ID, t.Family, t.Volume, params[i], "", "")
			}
		}
	}

	return out
}

func findPathKeywordType(types []DorkType) *DorkType {
	for i := range types {
		if strings.Contains(types[i].Pattern, "{kw}") {
			return &types[i]
		}
	}
	return nil
}

// cloneKWLimit and cloneParamLimit cap clone combinations.
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
