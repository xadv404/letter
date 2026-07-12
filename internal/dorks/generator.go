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

const maxPrecisionKeywords = 30

// GenerateFingerprint builds injectable-focused dorks: volume first, precision second.
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

	// Layer 1 — volume: param-only injectable surface (most Google results).
	for _, pm := range fp.Parameters {
		for _, d := range applyVolumeParam(pm) {
			add(d)
		}
		for _, d := range applyParamSurface(pm) {
			add(d)
		}
	}

	// Layer 2 — volume: crawled paths × injectable params.
	for _, path := range fp.Paths {
		for _, pm := range fp.Parameters {
			for _, d := range applyVolumePath(path, pm) {
				add(d)
			}
			for _, d := range applyPathLayout(path, pm) {
				add(d)
			}
		}
	}

	// Layer 3 — multi-param URLs (classic SQLi entry points).
	for _, d := range applyMultiParam(fp.Parameters) {
		add(d)
	}

	// Layer 4 — precision: theme keywords × params (clone hunting).
	kwLimit := len(fp.Keywords)
	if kwLimit > maxPrecisionKeywords {
		kwLimit = maxPrecisionKeywords
	}
	for i := 0; i < kwLimit; i++ {
		kw := fp.Keywords[i]
		for _, pm := range fp.Parameters {
			for _, d := range applyKeywordMatch(kw, pm) {
				add(d)
			}
			for _, d := range applySQLiDynamic(kw, pm) {
				add(d)
			}
		}
	}

	phLimit := len(fp.Phrases)
	if phLimit > 15 {
		phLimit = 15
	}
	for i := 0; i < phLimit; i++ {
		phrase := fp.Phrases[i]
		for _, pm := range fp.Parameters {
			for _, d := range applyPhraseClone(phrase, pm) {
				add(d)
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
	b.WriteString(fmt.Sprintf("Preview (%d dorks, %d templates):\n", len(dorks), TemplateCount()))
	for _, d := range dorks {
		b.WriteString("  - ")
		b.WriteString(d)
		b.WriteByte('\n')
	}
	return b.String()
}
