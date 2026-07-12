package dorks

import (
	"fmt"
	"strings"
)

// Options drives the keyword × parameter dork matrix (Phase 4).
// Host is the crawled seed (keywords/params source only); dorks target similar sites, not that host.
type Options struct {
	Host         string   // seed domain — not embedded in dorks
	TLD          string   // .com — scopes discovery to the same TLD family
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

// SiteScope extracts host and wildcard TLD from a domain/URL.
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
	if opts.TLD == "" && opts.Host != "" {
		_, opts.TLD = SiteScope(opts.Host)
	}
	if opts.TLD == "" || len(opts.Keywords) == 0 || len(opts.Parameters) == 0 {
		return nil
	}

	var out []string
	for _, kw := range opts.Keywords {
		for _, param := range opts.Parameters {
			for _, dork := range buildDorks(opts.TLD, kw, param) {
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

func buildDorks(tld, keyword, param string) []string {
	kw := strings.TrimSpace(keyword)
	pm := strings.TrimSpace(param)
	if kw == "" || pm == "" {
		return nil
	}

	quotedPM := pm + "="
	wildTLD := "site:*" + tld

	return []string{
		// Global — find similar vulnerable endpoints anywhere
		fmt.Sprintf(`inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:"%s"`, pm, kw),
		fmt.Sprintf(`inurl:"%s" "%s"`, quotedPM, kw),
		fmt.Sprintf(`intext:"%s" inurl:"%s"`, kw, quotedPM),
		fmt.Sprintf(`intext:%s inurl:%s=`, kw, pm),
		fmt.Sprintf(`inurl:%s= "%s"`, pm, kw),
		fmt.Sprintf(`allinurl:%s %s`, pm, kw),
		fmt.Sprintf(`inurl:%s intext:%s`, pm, kw),

		// Same TLD family — similar stacks, different hosts
		fmt.Sprintf(`%s inurl:%s= intext:%s`, wildTLD, pm, kw),
		fmt.Sprintf(`%s inurl:%s= intext:"%s"`, wildTLD, pm, kw),
		fmt.Sprintf(`%s inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s inurl:%s`, wildTLD, pm),
		fmt.Sprintf(`%s filetype:php inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s filetype:asp inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s filetype:jsp inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s filetype:cfm inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s filetype:php "%s"`, wildTLD, quotedPM),
		fmt.Sprintf(`%s inurl:%s filetype:php intext:%s`, wildTLD, pm, kw),
		fmt.Sprintf(`%s inurl:%s filetype:asp intext:%s`, wildTLD, pm, kw),
		fmt.Sprintf(`%s inurl:%s= ext:php`, wildTLD, pm),
		fmt.Sprintf(`%s inurl:%s= ext:asp`, wildTLD, pm),
		fmt.Sprintf(`inurl:%s intext:%s %s`, pm, kw, wildTLD),
		fmt.Sprintf(`"%s" %s inurl:%s=`, kw, wildTLD, pm),

		// Filetype + param (common SQLi stacks)
		fmt.Sprintf(`inurl:%s= intext:%s filetype:php`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:%s filetype:asp`, pm, kw),
		fmt.Sprintf(`filetype:php inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`filetype:asp inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`filetype:php inurl:%s=`, pm),
		fmt.Sprintf(`filetype:asp inurl:%s=`, pm),

		// Path context + injectable param
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

		fmt.Sprintf(`%s inurl:admin inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s inurl:login inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s inurl:search inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s inurl:product inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s inurl:view inurl:%s=`, wildTLD, pm),
		fmt.Sprintf(`%s inurl:api inurl:%s=`, wildTLD, pm),
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
	return len(buildDorks(".com", "admin", "id"))
}
