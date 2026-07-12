package dorks

import "strings"

// coreInjectable are always merged into dork params for volume SQLi hunting.
var coreInjectable = []string{
	"id", "pid", "cid", "nid", "gid", "tid", "uid", "sid",
	"cat", "catid", "cat_id", "category", "category_id",
	"product_id", "item_id", "article_id", "page_id", "user_id", "order_id",
	"search_term", "filter_by", "query", "q", "keyword", "search", "filter",
	"num", "idx", "ref", "code",
}

var classicInjectable = func() map[string]struct{} {
	m := make(map[string]struct{}, len(coreInjectable))
	for _, p := range coreInjectable {
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
	for _, p := range coreInjectable {
		add(p)
	}
	return out
}
