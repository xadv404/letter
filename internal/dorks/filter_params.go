package dorks

import (
	"regexp"
	"sort"
	"strings"
)

const (
	MaxAssembledDorks        = 180
	MaxParamsAssemble        = 15
	MaxVolumeDorksPerParam   = 2
	MaxErrorDorksPerParam    = 1
	MaxKeywordTemplatesPerKW = 2
)

var noiseParamPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^_`),
	regexp.MustCompile(`(?i)^(gl|gclid|fbclid|msclkid|utm_|mc_|pk_|_ga|_gid)`),
	regexp.MustCompile(`(?i)(contextual|player_version|playlist|programslug|samlrequest|service_type|service_version|signup|selectedprice|fallback|cachebuster)`),
	regexp.MustCompile(`(?i)^(co|ct|pt|lng|hl|lang|locale|currency|timezone|format|output|callback|jsonp|ajax|ver|version|rev|build|v|ts|rand|cb)$`),
	regexp.MustCompile(`(?i)^(gift|event|ref|referrer|referer|source|medium|campaign|content|term|redirect|next|return)$`),
	regexp.MustCompile(`(?i)^(width|height|color|theme|skin|font|sidebar|widget|menu|nav|module|plugin)$`),
}

// coreVolumeTypeIDs — broad patterns only (no per-script spam).
var coreVolumeTypeIDs = []string{"v01", "v02", "v04"}

// selectParamsForAssembly keeps injectable, high-score params — drops tracking noise.
func selectParamsForAssembly(m Materials) []string {
	type ranked struct {
		name  string
		score int
	}
	var items []ranked
	seen := map[string]struct{}{}
	for _, p := range m.Params {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" || isNoiseParam(p) {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		ps := m.ParamScores[p]
		if ps == 0 && !IsExploitableParam(p) && !isPrimaryInjectable(p) {
			continue
		}
		if ps > 0 && ps < 50 && !IsExploitableParam(p) {
			continue
		}
		items = append(items, ranked{name: p, score: ps})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].score == items[j].score {
			return items[i].name < items[j].name
		}
		return items[i].score > items[j].score
	})
	out := make([]string, 0, MaxParamsAssemble)
	for _, it := range items {
		out = append(out, it.name)
		if len(out) >= MaxParamsAssemble {
			break
		}
	}
	if len(out) == 0 {
		return []string{"id", "cat", "pid", "product_id", "item_id"}
	}
	return out
}

func isNoiseParam(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, re := range noiseParamPatterns {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// volumeTypesForParam picks at most MaxVolumeDorksPerParam non-redundant volume templates.
func volumeTypesForParam(pm string, paths []string, typesByID map[string]DorkType) []DorkType {
	var out []DorkType
	addID := func(id string) {
		if len(out) >= MaxVolumeDorksPerParam {
			return
		}
		if t, ok := typesByID[id]; ok {
			for _, existing := range out {
				if existing.ID == id {
					return
				}
			}
			out = append(out, t)
		}
	}
	addID("v01")
	if IsExploitableParam(pm) || isPrimaryInjectable(pm) {
		addID("v02")
	}
	if scriptID := scriptTypeForPaths(paths); scriptID != "" {
		addID(scriptID)
	}
	return out
}

func scriptTypeForPaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	scriptToType := map[string]string{
		"product": "v07", "view": "v06", "news": "v08", "index": "v09",
		"article": "v10", "detail": "v11", "catalog": "v14", "category": "v15",
		"item": "v16", "list": "v17", "forum": "v18", "shop": "v13",
	}
	for _, path := range paths {
		path = strings.ToLower(strings.Trim(path, "/"))
		seg := path
		if i := strings.LastIndex(path, "/"); i >= 0 {
			seg = path[i+1:]
		}
		if id, ok := scriptToType[seg]; ok {
			return id
		}
	}
	return ""
}

func typesByID(types []DorkType) map[string]DorkType {
	m := make(map[string]DorkType, len(types))
	for _, t := range types {
		m[t.ID] = t
	}
	return m
}

func pickKeywordTypes(types []DorkType, limit int) []DorkType {
	prefer := []string{"k01", "k07", "k02", "k06", "k08"}
	byID := typesByID(types)
	var out []DorkType
	for _, id := range prefer {
		if t, ok := byID[id]; ok {
			out = append(out, t)
			if len(out) >= limit {
				return out
			}
		}
	}
	for _, t := range types {
		if len(out) >= limit {
			break
		}
		out = append(out, t)
	}
	return out
}

func pickErrorTypes(types []DorkType) []DorkType {
	prefer := []string{"e01", "e02"}
	byID := typesByID(types)
	var out []DorkType
	for _, id := range prefer {
		if t, ok := byID[id]; ok {
			out = append(out, t)
		}
	}
	return out
}
