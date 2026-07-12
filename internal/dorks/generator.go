package dorks

import (
	"fmt"
	"strings"
)

// Options drives the keyword × parameter dork matrix (Phase 4).
// Host is the crawled seed (keywords/params source only); dorks are global searches.
type Options struct {
	Host         string   // seed domain — not embedded in dorks
	TLD          string   // extracted from seed (logging only)
	Keywords     []string // Phase 2 output
	Parameters   []string // Phase 3 HIGH/MEDIUM params only
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

func (g *Generator) Generate(opts Options) []string {
	if len(opts.Keywords) == 0 || len(opts.Parameters) == 0 {
		return nil
	}

	var out []string
	for _, kw := range opts.Keywords {
		for _, param := range opts.Parameters {
			for _, dork := range buildDorks(kw, param) {
				if _, ok := g.seen[dork]; ok {
					continue
				}
				g.seen[dork] = struct{}{}
				out = append(out, dork)
				if opts.PreviewLimit > 0 && len(out) >= opts.PreviewLimit {
					return out
				}
			}
		}
	}
	return out
}

func buildDorks(keyword, param string) []string {
	kw := strings.TrimSpace(keyword)
	pm := strings.TrimSpace(param)
	if kw == "" || pm == "" {
		return nil
	}

	quotedPM := pm + "="

	return []string{
		fmt.Sprintf(`inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:"%s"`, pm, kw),
		fmt.Sprintf(`inurl:"%s" "%s"`, quotedPM, kw),
		fmt.Sprintf(`intext:"%s" inurl:"%s"`, kw, quotedPM),
		fmt.Sprintf(`intext:%s inurl:%s=`, kw, pm),
		fmt.Sprintf(`inurl:%s= "%s"`, pm, kw),
		fmt.Sprintf(`allinurl:%s %s`, pm, kw),
		fmt.Sprintf(`inurl:%s intext:%s`, pm, kw),

		fmt.Sprintf(`inurl:%s= intext:%s filetype:php`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:%s filetype:asp`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:%s filetype:jsp`, pm, kw),
		fmt.Sprintf(`filetype:php inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`filetype:asp inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`filetype:jsp inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`filetype:php inurl:%s=`, pm),
		fmt.Sprintf(`filetype:asp inurl:%s=`, pm),
		fmt.Sprintf(`filetype:jsp inurl:%s=`, pm),

		fmt.Sprintf(`inurl:admin inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:login inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:search inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:product inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:view inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:api inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:report inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:download inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:page inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:index inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:users inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:member inurl:%s= intext:%s`, pm, kw),
	}
}

func Preview(opts Options, limit int) string {
	opts.PreviewLimit = limit
	g := New()
	dorks := g.Generate(opts)
	if len(dorks) == 0 {
		return "No dorks generated."
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

// TemplateCount returns the number of dork patterns per keyword×param pair.
func TemplateCount() int {
	return len(buildDorks("admin", "id"))
}
