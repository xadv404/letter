package dorks

import (
	"fmt"
	"strings"
)

// Options drives the keyword × parameter dork matrix (Phase 4).
type Options struct {
	Host         string   // example.com
	TLD          string   // .com
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
	if opts.Host == "" {
		return nil
	}
	if opts.TLD == "" {
		_, opts.TLD = SiteScope(opts.Host)
	}

	var out []string
	for _, kw := range opts.Keywords {
		for _, param := range opts.Parameters {
			for _, dork := range buildDorks(opts.Host, opts.TLD, kw, param) {
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

func buildDorks(host, tld, keyword, param string) []string {
	kw := strings.TrimSpace(keyword)
	pm := strings.TrimSpace(param)
	if kw == "" || pm == "" {
		return nil
	}

	quotedPM := pm + "="

	return []string{
		// Core matrix (user spec)
		fmt.Sprintf(`inurl:%s= intext:%s`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:%s site:%s`, pm, kw, host),
		fmt.Sprintf(`inurl:%s= intext:%s site:%s`, pm, kw, tld),
		fmt.Sprintf(`inurl:"%s" "%s"`, quotedPM, kw),
		fmt.Sprintf(`intext:"%s" inurl:"%s"`, kw, quotedPM),
		fmt.Sprintf(`intext:%s inurl:%s=`, kw, pm),

		// Filetype variants
		fmt.Sprintf(`site:%s filetype:php inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s filetype:asp inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s filetype:jsp inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s filetype:cfm inurl:%s=`, host, pm),
		fmt.Sprintf(`site:*%s filetype:php "%s"`, tld, quotedPM),

		// Wildcard + intext
		fmt.Sprintf(`site:*%s inurl:%s= intext:%s`, tld, pm, kw),
		fmt.Sprintf(`site:*%s inurl:%s= intext:"%s"`, tld, pm, kw),

		// inurl-only recon
		fmt.Sprintf(`site:%s inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s inurl:%s`, host, pm),
		fmt.Sprintf(`inurl:%s= "%s"`, pm, kw),

		// Contextual paths + param
		fmt.Sprintf(`site:%s inurl:admin inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s inurl:login inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s inurl:search inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s inurl:product inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s inurl:view inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s inurl:api inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s inurl:report inurl:%s=`, host, pm),
		fmt.Sprintf(`site:%s inurl:download inurl:%s=`, host, pm),

		// intext quoted phrases
		fmt.Sprintf(`inurl:%s= intext:"%s"`, pm, kw),
		fmt.Sprintf(`intext:"%s" inurl:%s= site:%s`, kw, pm, host),
		fmt.Sprintf(`site:%s intext:%s inurl:%s=`, host, kw, pm),
		fmt.Sprintf(`site:%s "%s" inurl:%s=`, host, kw, pm),
		fmt.Sprintf(`inurl:%s intext:%s site:%s`, pm, kw, host),
		fmt.Sprintf(`allinurl:%s %s site:%s`, pm, kw, host),
		fmt.Sprintf(`site:%s inurl:%s= | inurl:%s intext:%s`, host, pm, pm, kw),
		fmt.Sprintf(`site:*%s inurl:%s filetype:php intext:%s`, tld, pm, kw),
		fmt.Sprintf(`site:*%s inurl:%s filetype:asp intext:%s`, tld, pm, kw),
		fmt.Sprintf(`site:%s inurl:%s= ext:php`, host, pm),
		fmt.Sprintf(`site:%s inurl:%s= ext:asp`, host, pm),
		fmt.Sprintf(`inurl:%s= intext:%s filetype:php`, pm, kw),
		fmt.Sprintf(`inurl:%s= intext:%s filetype:asp`, pm, kw),
	}
}

func quoteIfSpace(s string) string {
	if strings.Contains(s, " ") {
		return s
	}
	return s
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
	return len(buildDorks("example.com", ".com", "admin", "id"))
}
