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

const maxPrecisionKeywords = 5

// GenerateFingerprint builds injectable-focused dorks optimized for max Google volume.
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

	primary, allParams := splitParamTiers(fp.Parameters)

	// Layer 0 — global blast: fixed literal patterns (highest Google yield).
	for _, d := range globalLiteralBlast() {
		add(d)
	}

	// Layer 1 — ultra-simple: every param, single-operator dorks.
	for _, pm := range allParams {
		for _, d := range applyUltraSimple(pm) {
			add(d)
		}
	}

	// Layer 2 — literal endpoints: script.php?param= (single inurl, injectable).
	for _, pm := range allParams {
		for _, d := range applyLiteralEndpoints(pm) {
			add(d)
		}
	}

	// Layer 3 — injectable errors + stack filetype (primary params).
	for _, pm := range primary {
		for _, d := range applyInjectableErrors(pm) {
			add(d)
		}
		for _, d := range applyStackFiletype(pm) {
			add(d)
		}
		for _, d := range applyParamSurface(pm) {
			add(d)
		}
	}

	// Layer 4 — multi-param combos (classic SQLi entry points).
	for _, d := range applyMultiParam(primary) {
		add(d)
	}

	// Layer 5 — crawled paths × primary params (capped).
	for _, path := range fp.TopPaths() {
		for _, pm := range primary {
			for _, d := range applySimplePath(path, pm) {
				add(d)
			}
		}
	}

	// Layer 6 — precision clone hunting (minimal — low Google volume per dork).
	kwLimit := len(fp.Keywords)
	if kwLimit > maxPrecisionKeywords {
		kwLimit = maxPrecisionKeywords
	}
	for i := 0; i < kwLimit; i++ {
		kw := fp.Keywords[i]
		for _, pm := range primary {
			for _, d := range applyKeywordMatchLite(kw, pm) {
				add(d)
			}
		}
	}

	phLimit := len(fp.Phrases)
	if phLimit > 5 {
		phLimit = 5
	}
	for i := 0; i < phLimit; i++ {
		for _, pm := range primary {
			for _, d := range applyPhraseClone(fp.Phrases[i], pm) {
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
