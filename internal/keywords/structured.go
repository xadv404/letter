package keywords

import (
	"encoding/json"
	"strings"

	"golang.org/x/net/html"
)

// collectStructured extracts OpenGraph, JSON-LD and anchor-rich signals.
func collectStructured(doc *html.Node) []block {
	var blocks []block
	blocks = append(blocks, collectMeta(doc)...)
	blocks = append(blocks, collectJSONLD(doc)...)
	blocks = append(blocks, collectAnchorTexts(doc)...)
	return blocks
}

func collectJSONLD(doc *html.Node) []block {
	var blocks []block
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			var typ, content string
			for _, a := range n.Attr {
				if strings.ToLower(a.Key) == "type" {
					typ = strings.ToLower(a.Val)
				}
			}
			if typ == "application/ld+json" && n.FirstChild != nil {
				content = strings.TrimSpace(n.FirstChild.Data)
			}
			if content != "" {
				for _, text := range jsonLDStrings(content) {
					if text != "" {
						blocks = append(blocks, block{tag: "jsonld", weight: 4, text: text})
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return blocks
}

func jsonLDStrings(raw string) []string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	var walk func(any)
	walk = func(node any) {
		switch t := node.(type) {
		case map[string]any:
			for k, val := range t {
				lk := strings.ToLower(k)
				if lk == "name" || lk == "description" || lk == "keywords" || lk == "headline" || lk == "alternatename" {
					if s, ok := val.(string); ok {
						s = strings.TrimSpace(s)
						if s != "" {
							if _, ok := seen[s]; !ok {
								seen[s] = struct{}{}
								out = append(out, s)
							}
						}
					}
				}
				walk(val)
			}
		case []any:
			for _, item := range t {
				walk(item)
			}
		}
	}
	walk(v)
	return out
}

func collectAnchorTexts(doc *html.Node) []block {
	var blocks []block
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			text := strings.TrimSpace(deepText(n))
			if len(text) >= 3 && len(text) <= 120 {
				blocks = append(blocks, block{tag: "a", weight: 2, text: text})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return blocks
}
