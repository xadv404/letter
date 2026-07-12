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

// GenerateFingerprint builds dorks from a theme fingerprint using 50 templates × 6 families.
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

	for _, phrase := range fp.Phrases {
		for _, pm := range fp.Parameters {
			for _, d := range applyPhraseClone(phrase, pm) {
				add(d)
			}
		}
	}

	for _, kw := range fp.Keywords {
		for _, pm := range fp.Parameters {
			for _, d := range applyKeywordMatch(kw, pm) {
				add(d)
			}
		}
	}

	for _, path := range fp.Paths {
		for _, pm := range fp.Parameters {
			for _, d := range applyPathLayout(path, pm) {
				add(d)
			}
		}
	}

	for _, pm := range fp.Parameters {
		for _, d := range applyParamSurface(pm) {
			add(d)
		}
	}

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
	b.WriteString(fmt.Sprintf("Preview (%d dorks, %d templates × 6 families):\n", len(dorks), TemplateCount()))
	for _, d := range dorks {
		b.WriteString("  - ")
		b.WriteString(d)
		b.WriteByte('\n')
	}
	return b.String()
}
