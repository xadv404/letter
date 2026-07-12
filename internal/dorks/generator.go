package dorks

import (
	"fmt"
	"strings"
)

// Options drives the keyword × parameter dork matrix (legacy API).
type Options struct {
	Host         string
	TLD          string
	Keywords     []string
	Parameters   []string
	PreviewLimit int
}

type Generator struct {
	seen map[string]struct{}
}

func New() *Generator {
	return &Generator{seen: map[string]struct{}{}}
}

// SiteScope extracts host and TLD from a domain/URL.
func SiteScope(raw string) (host, tld string) {
	raw = strings.TrimPrefix(strings.TrimSpace(raw), "https://")
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimSuffix(raw, "/")
	if i := strings.Index(raw, "/"); i >= 0 {
		raw = raw[:i]
	}
	host = raw
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		tld = "." + parts[len(parts)-1]
	}
	return host, tld
}

// GenerateFingerprint builds a maximal set of SQLi-oriented dorks from a theme fingerprint.
func (g *Generator) GenerateFingerprint(fp Fingerprint) []string {
	if !fp.Viable() {
		return nil
	}
	var out []string
	add := func(dork string) {
		dork = strings.TrimSpace(dork)
		if dork == "" {
			return
		}
		if _, ok := g.seen[dork]; ok {
			return
		}
		g.seen[dork] = struct{}{}
		out = append(out, dork)
	}

	// 1) Exact phrases × params — best clone hunters (same template/stack).
	for _, phrase := range fp.Phrases {
		for _, pm := range fp.Parameters {
			for _, d := range phraseParamDorks(phrase, pm) {
				add(d)
			}
		}
	}

	// 2) Keywords × params — broad similar-site discovery.
	for _, kw := range fp.Keywords {
		for _, pm := range fp.Parameters {
			for _, d := range keywordParamDorks(kw, pm) {
				add(d)
			}
		}
	}

	// 3) Path structure × params — same URL layout, unknown host.
	for _, path := range fp.Paths {
		for _, pm := range fp.Parameters {
			for _, d := range pathParamDorks(path, pm) {
				add(d)
			}
		}
	}

	// 4) Param-only + filetype — wide SQLi surface.
	for _, pm := range fp.Parameters {
		for _, d := range paramOnlyDorks(pm) {
			add(d)
		}
	}

	// 5) Cross-param combos on high-signal paths.
	if len(fp.Parameters) >= 2 {
		for i := 0; i < len(fp.Parameters) && i < 12; i++ {
			for j := i + 1; j < len(fp.Parameters) && j < i+4; j++ {
				add(fmt.Sprintf(`inurl:%s= inurl:%s=`, fp.Parameters[i], fp.Parameters[j]))
				add(fmt.Sprintf(`allinurl:%s %s`, fp.Parameters[i], fp.Parameters[j]))
			}
		}
	}

	return out
}

func (g *Generator) Generate(opts Options) []string {
	fp := NewFingerprint()
	for _, kw := range opts.Keywords {
		fp.AddTerm(kw)
	}
	for _, pm := range opts.Parameters {
		fp.AddParameter(pm)
	}
	fp.Finalize()
	out := g.GenerateFingerprint(*fp)
	if opts.PreviewLimit > 0 && len(out) > opts.PreviewLimit {
		return out[:opts.PreviewLimit]
	}
	return out
}

func phraseParamDorks(phrase, param string) []string {
	pm := strings.TrimSpace(param)
	ph := strings.TrimSpace(phrase)
	if pm == "" || ph == "" {
		return nil
	}
	qpm := pm + "="
	return []string{
		fmt.Sprintf(`intext:"%s" inurl:%s=`, ph, pm),
		fmt.Sprintf(`inurl:%s= intext:"%s"`, pm, ph),
		fmt.Sprintf(`intext:"%s" inurl:"%s"`, ph, qpm),
		fmt.Sprintf(`inurl:"%s" "%s"`, qpm, ph),
		fmt.Sprintf(`intext:"%s" filetype:php inurl:%s=`, ph, pm),
		fmt.Sprintf(`intext:"%s" filetype:asp inurl:%s=`, ph, pm),
		fmt.Sprintf(`intext:"%s" inurl:%s= filetype:php`, ph, pm),
	}
}

func keywordParamDorks(keyword, param string) []string {
	kw := strings.TrimSpace(keyword)
	pm := strings.TrimSpace(param)
	if kw == "" || pm == "" {
		return nil
	}
	qpm := pm + "="
	return []string{
		fmt.Sprintf(`inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:"%s"`, pm, kw),
		fmt.Sprintf(`inurl:"%s" "%s"`, qpm, kw),
		fmt.Sprintf(`intext:"%s" inurl:"%s"`, kw, qpm),
		fmt.Sprintf(`intext:%s inurl:%s=`, kw, pm),
		fmt.Sprintf(`inurl:%s= "%s"`, pm, kw),
		fmt.Sprintf(`allinurl:%s %s`, pm, kw),
		fmt.Sprintf(`inurl:%s intext:%s`, pm, kw),

		fmt.Sprintf(`inurl:%s= intext:%s filetype:php`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:%s filetype:asp`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:%s filetype:jsp`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:%s filetype:aspx`, pm, kw),
		fmt.Sprintf(`filetype:php inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`filetype:asp inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`filetype:php inurl:%s=`, pm),
		fmt.Sprintf(`filetype:asp inurl:%s=`, pm),
		fmt.Sprintf(`filetype:jsp inurl:%s=`, pm),

		fmt.Sprintf(`inurl:admin inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:login inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:search inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:product inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:view inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:item inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:detail inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:list inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:page inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:module inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:include inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:content inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:download inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:report inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:api inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:ajax inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:show inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:display inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:read inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:process inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:action inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:cart inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:checkout inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:order inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:catalog inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:category inurl:%s= intext:%s`, pm, kw),
	}
}

func pathParamDorks(path, param string) []string {
	path = strings.Trim(path, "/")
	pm := strings.TrimSpace(param)
	if path == "" || pm == "" {
		return nil
	}
	return []string{
		fmt.Sprintf(`inurl:%s inurl:%s=`, path, pm),
		fmt.Sprintf(`allinurl:%s %s`, path, pm),
		fmt.Sprintf(`inurl:%s filetype:php inurl:%s=`, path, pm),
		fmt.Sprintf(`inurl:%s filetype:asp inurl:%s=`, path, pm),
		fmt.Sprintf(`inurl:%s.php inurl:%s=`, strings.TrimSuffix(path, ".php"), pm),
	}
}

func paramOnlyDorks(param string) []string {
	pm := strings.TrimSpace(param)
	if pm == "" {
		return nil
	}
	qpm := pm + "="
	return []string{
		fmt.Sprintf(`inurl:%s=`, pm),
		fmt.Sprintf(`inurl:"%s"`, qpm),
		fmt.Sprintf(`inurl:%s filetype:php`, pm),
		fmt.Sprintf(`inurl:%s filetype:asp`, pm),
		fmt.Sprintf(`inurl:%s filetype:jsp`, pm),
		fmt.Sprintf(`filetype:php inurl:%s=`, pm),
		fmt.Sprintf(`filetype:asp inurl:%s=`, pm),
		fmt.Sprintf(`inurl:admin inurl:%s=`, pm),
		fmt.Sprintf(`inurl:login inurl:%s=`, pm),
		fmt.Sprintf(`inurl:search inurl:%s=`, pm),
		fmt.Sprintf(`inurl:index.php inurl:%s=`, pm),
		fmt.Sprintf(`inurl:view.php inurl:%s=`, pm),
		fmt.Sprintf(`inurl:product.php inurl:%s=`, pm),
		fmt.Sprintf(`inurl:page.php inurl:%s=`, pm),
	}
}

func Preview(opts Options, limit int) string {
	return PreviewList(New().Generate(opts), limit)
}

func PreviewList(dorks []string, limit int) string {
	if len(dorks) == 0 {
		return "No dorks generated."
	}
	if limit > 0 && len(dorks) > limit {
		dorks = dorks[:limit]
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Preview (%d dorks):\n", len(dorks)))
	for _, d := range dorks {
		b.WriteString("  - ")
		b.WriteString(d)
		b.WriteByte('\n')
	}
	return b.String()
}

// TemplateCount returns dork patterns per keyword×param pair.
func TemplateCount() int {
	return len(keywordParamDorks("admin", "id"))
}

// buildDorks is kept for tests referencing the matrix size.
func buildDorks(keyword, param string) []string {
	return keywordParamDorks(keyword, param)
}
