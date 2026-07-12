package dorks

import (
	"fmt"
	"strings"
)

// Slot placeholders used in dork type patterns.
const (
	SlotParam = "param"
	SlotKW    = "kw"
	SlotPath  = "path"
)

// DorkType is a reusable Google dork pattern — combine with keywords/params externally.
type DorkType struct {
	ID      string
	Family  string
	Pattern string
	Slots   []string
	Volume  string // high = max URLs, medium = clone precision
}

// Materials is the crawl output: types + keywords + params (no assembled dorks).
type Materials struct {
	Types    []DorkType
	Keywords []string
	Phrases  []string
	Params   []string
	Paths    []string
}

// AllDorkTypes returns curated dork type templates (~45 patterns).
func AllDorkTypes() []DorkType {
	return []DorkType{
		// ── Volume: param only (1 slot) — max Google URLs ──
		{ID: "v01", Family: "param_volume", Pattern: `inurl:{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v02", Family: "param_volume", Pattern: `inurl:.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v03", Family: "param_volume", Pattern: `inurl:.asp?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v04", Family: "param_volume", Pattern: `inurl:?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v05", Family: "param_volume", Pattern: `inurl:&{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v06", Family: "param_volume", Pattern: `inurl:view.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v07", Family: "param_volume", Pattern: `inurl:product.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v08", Family: "param_volume", Pattern: `inurl:news.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v09", Family: "param_volume", Pattern: `inurl:index.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v10", Family: "param_volume", Pattern: `inurl:article.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v11", Family: "param_volume", Pattern: `inurl:detail.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v12", Family: "param_volume", Pattern: `inurl:show.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v13", Family: "param_volume", Pattern: `inurl:shop.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v14", Family: "param_volume", Pattern: `inurl:catalog.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v15", Family: "param_volume", Pattern: `inurl:category.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v16", Family: "param_volume", Pattern: `inurl:item.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v17", Family: "param_volume", Pattern: `inurl:list.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v18", Family: "param_volume", Pattern: `inurl:forum.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v19", Family: "param_volume", Pattern: `inurl:profile.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},
		{ID: "v20", Family: "param_volume", Pattern: `inurl:download.php?{param}=`, Slots: []string{SlotParam}, Volume: "high"},

		// ── Clone: keyword × param (2 slots) — find similar sites ──
		{ID: "k01", Family: "keyword_match", Pattern: `inurl:{param}= intext:{kw}`, Slots: []string{SlotParam, SlotKW}, Volume: "medium"},
		{ID: "k02", Family: "keyword_match", Pattern: `intext:"{kw}" inurl:{param}=`, Slots: []string{SlotKW, SlotParam}, Volume: "medium"},
		{ID: "k03", Family: "keyword_match", Pattern: `inurl:{param}= intext:"{kw}"`, Slots: []string{SlotParam, SlotKW}, Volume: "medium"},
		{ID: "k04", Family: "keyword_match", Pattern: `inurl:"{param}=" "{kw}"`, Slots: []string{SlotParam, SlotKW}, Volume: "medium"},
		{ID: "k05", Family: "keyword_match", Pattern: `inurl:view.php?{param}= intext:{kw}`, Slots: []string{SlotParam, SlotKW}, Volume: "medium"},
		{ID: "k06", Family: "keyword_match", Pattern: `inurl:product.php?{param}= intext:{kw}`, Slots: []string{SlotParam, SlotKW}, Volume: "medium"},
		{ID: "k07", Family: "keyword_match", Pattern: `inurl:.php?{param}= intext:{kw}`, Slots: []string{SlotParam, SlotKW}, Volume: "medium"},
		{ID: "k08", Family: "keyword_match", Pattern: `filetype:php inurl:{param}= intext:{kw}`, Slots: []string{SlotParam, SlotKW}, Volume: "medium"},
		{ID: "k09", Family: "keyword_match", Pattern: `allinurl:{param} {kw}`, Slots: []string{SlotParam, SlotKW}, Volume: "medium"},
		{ID: "k10", Family: "keyword_match", Pattern: `"{kw}" inurl:{param}=`, Slots: []string{SlotKW, SlotParam}, Volume: "medium"},

		// ── Path context: path × param × keyword (3 slots) ──
		{ID: "p01", Family: "path_context", Pattern: `inurl:{path} inurl:{param}= intext:{kw}`, Slots: []string{SlotPath, SlotParam, SlotKW}, Volume: "medium"},
		{ID: "p02", Family: "path_context", Pattern: `inurl:{path}.php?{param}= intext:{kw}`, Slots: []string{SlotPath, SlotParam, SlotKW}, Volume: "medium"},
		{ID: "p03", Family: "path_context", Pattern: `inurl:{path}?{param}=`, Slots: []string{SlotPath, SlotParam}, Volume: "high"},
		{ID: "p04", Family: "path_context", Pattern: `inurl:{path}.php?{param}=`, Slots: []string{SlotPath, SlotParam}, Volume: "high"},

		// ── Error leak: already broken DB (param slot) ──
		{ID: "e01", Family: "sqli_error", Pattern: `intext:"You have an error in your SQL syntax" inurl:{param}=`, Slots: []string{SlotParam}, Volume: "medium"},
		{ID: "e02", Family: "sqli_error", Pattern: `intext:"mysql_fetch" inurl:{param}=`, Slots: []string{SlotParam}, Volume: "medium"},
		{ID: "e03", Family: "sqli_error", Pattern: `intext:"SQL syntax" inurl:view.php?{param}=`, Slots: []string{SlotParam}, Volume: "medium"},
		{ID: "e04", Family: "sqli_error", Pattern: `intext:"mysql_fetch" inurl:product.php?{param}=`, Slots: []string{SlotParam}, Volume: "medium"},

		// ── Multi-param surface ──
		{ID: "m01", Family: "multi_param", Pattern: `inurl:{param}= inurl:{param2}=`, Slots: []string{SlotParam, "param2"}, Volume: "medium"},
	}
}

// DorkTypeCount returns the number of curated templates.
func DorkTypeCount() int { return len(AllDorkTypes()) }

// PrepareMaterials builds export kit from crawl fingerprint + ranked keywords.
func PrepareMaterials(fp Fingerprint, keywords, phrases []string) Materials {
	types := AllDorkTypes()
	params := capStrings(fp.Parameters, 30)
	paths := fp.TopPaths()
	if len(paths) > 15 {
		paths = paths[:15]
	}
	kw := capStrings(keywords, 60)
	ph := capStrings(phrases, 20)
	return Materials{
		Types:    types,
		Keywords: kw,
		Phrases:  ph,
		Params:   params,
		Paths:    paths,
	}
}

// PreviewMaterials formats a human-readable summary.
func PreviewMaterials(m Materials) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Dork types: %d | Keywords: %d | Phrases: %d | Params: %d | Paths: %d\n",
		len(m.Types), len(m.Keywords), len(m.Phrases), len(m.Params), len(m.Paths))
	b.WriteString("Auto-assemble: types × keywords × params → dorks.txt\n")
	b.WriteString("Example types:\n")
	for i, t := range m.Types {
		if i >= 5 {
			break
		}
		b.WriteString("  ")
		b.WriteString(t.Pattern)
		b.WriteByte('\n')
	}
	if len(m.Keywords) > 0 {
		n := 8
		if len(m.Keywords) < n {
			n = len(m.Keywords)
		}
		b.WriteString("Top keywords: ")
		b.WriteString(strings.Join(m.Keywords[:n], ", "))
		b.WriteByte('\n')
	}
	return b.String()
}
