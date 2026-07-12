package dorks

import (
	"fmt"
	"strings"
)

var templates = []string{
	`site:{domain} inurl:{param}`,
	`site:{domain} inurl:{param}=`,
	`site:{domain} intext:"{keyword}" inurl:{param}`,
	`site:{domain} inurl:{param} intext:"{keyword}"`,
	`site:{domain} inurl:{param} filetype:php`,
	`site:{domain} inurl:{param} filetype:asp`,
	`site:{domain} inurl:{param} filetype:jsp`,
	`site:{domain} inurl:{param} filetype:cfm`,
	`site:{domain} inurl:admin inurl:{param}`,
	`site:{domain} inurl:login inurl:{param}`,
	`site:{domain} inurl:api inurl:{param}`,
	`site:{domain} inurl:search inurl:{param}`,
	`site:{domain} inurl:product inurl:{param}`,
	`site:{domain} inurl:view inurl:{param}`,
	`site:{domain} inurl:page inurl:{param}`,
	`site:{domain} inurl:index inurl:{param}`,
	`site:{domain} inurl:detail inurl:{param}`,
	`site:{domain} inurl:profile inurl:{param}`,
	`site:{domain} inurl:report inurl:{param}`,
	`site:{domain} inurl:download inurl:{param}`,
	`site:{domain} inurl:upload inurl:{param}`,
	`site:{domain} inurl:cart inurl:{param}`,
	`site:{domain} inurl:checkout inurl:{param}`,
	`site:{domain} inurl:order inurl:{param}`,
	`site:{domain} inurl:invoice inurl:{param}`,
	`site:{domain} inurl:filter inurl:{param}`,
	`site:{domain} inurl:sort inurl:{param}`,
	`site:{domain} inurl:category inurl:{param}`,
	`site:{domain} inurl:archive inurl:{param}`,
	`site:{domain} inurl:news inurl:{param}`,
	`site:{domain} inurl:blog inurl:{param}`,
	`site:{domain} inurl:forum inurl:{param}`,
	`site:{domain} inurl:gallery inurl:{param}`,
	`site:{domain} inurl:media inurl:{param}`,
	`site:{domain} inurl:config inurl:{param}`,
	`site:{domain} inurl:backup inurl:{param}`,
	`site:{domain} inurl:test inurl:{param}`,
}

type Generator struct {
	seen map[string]struct{}
}

func New() *Generator {
	return &Generator{seen: map[string]struct{}{}}
}

func (g *Generator) Generate(domain string, keywords, parameters []string, previewLimit int) []string {
	var out []string
	for _, kw := range keywords {
		for _, param := range parameters {
			for _, tpl := range templates {
				dork := strings.NewReplacer(
					"{domain}", domain,
					"{keyword}", kw,
					"{param}", param,
				).Replace(tpl)
				if _, ok := g.seen[dork]; ok {
					continue
				}
				g.seen[dork] = struct{}{}
				out = append(out, dork)
				if previewLimit > 0 && len(out) >= previewLimit {
					return out
				}
			}
		}
	}
	return out
}

func Preview(domain string, keywords, parameters []string, limit int) string {
	g := New()
	dorks := g.Generate(domain, keywords, parameters, limit)
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
