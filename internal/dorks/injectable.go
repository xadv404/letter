package dorks

import "strings"

var classicInjectable = func() map[string]struct{} {
	m := make(map[string]struct{}, len(burpTopInjectable))
	for _, p := range burpTopInjectable {
		m[p] = struct{}{}
	}
	return m
}()

// IsInjectableParam returns true when a parameter name is worth SQLi dork hunting.
func IsInjectableParam(name string, score int) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	if score >= 65 {
		return true
	}
	if _, ok := classicInjectable[name]; ok {
		return true
	}
	if score >= 50 {
		for _, suf := range []string{"_id", "_num", "_no", "_code", "id", "num"} {
			if strings.HasSuffix(name, suf) || strings.Contains(name, suf) {
				return true
			}
		}
	}
	return false
}

// ExpandInjectableParams derives sibling/variant names from crawled parameters.
func ExpandInjectableParams(params []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" || len(p) > 48 {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, p := range params {
		add(p)
		if strings.HasSuffix(p, "_id") {
			add(strings.TrimSuffix(p, "_id"))
			add("id")
		}
		if strings.HasSuffix(p, "_num") {
			add(strings.TrimSuffix(p, "_num"))
			add("num")
		}
		if strings.Contains(p, "cat") {
			add("cat")
			add("catid")
			add("cat_id")
			add("category")
			add("category_id")
		}
		if strings.Contains(p, "product") {
			add("product")
			add("product_id")
			add("pid")
		}
		if strings.Contains(p, "article") {
			add("article")
			add("article_id")
			add("aid")
		}
		if strings.Contains(p, "item") {
			add("item")
			add("item_id")
		}
		if strings.Contains(p, "user") {
			add("user")
			add("user_id")
			add("uid")
		}
		if strings.Contains(p, "search") {
			add("search")
			add("search_term")
			add("q")
			add("query")
		}
	}
	for _, p := range burpTopInjectable {
		add(p)
	}
	return out
}

// PrioritizeInjectable sorts params: injectable first, capped at max.
func PrioritizeInjectable(params []string, max int) []string {
	if max <= 0 || len(params) <= max {
		return params
	}
	var primary, rest []string
	seen := map[string]struct{}{}
	for _, p := range params {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		if isPrimaryInjectable(p) {
			primary = append(primary, p)
		} else {
			rest = append(rest, p)
		}
	}
	out := append(primary, rest...)
	if len(out) > max {
		out = out[:max]
	}
	return out
}
