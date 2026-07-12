package params

import "strings"

// Category classifies URL parameters across 8 vulnerability families.
type Category string

const (
	CatSQLi      Category = "sqli"
	CatXSS       Category = "xss"
	CatRedirect  Category = "open_redirect"
	CatIDOR      Category = "idor"
	CatLFI       Category = "lfi"
	CatSSRF      Category = "ssrf"
	CatCmdInject Category = "command_injection"
	CatNoise     Category = "noise"
)

// categoryWeight scales the base score for dork prioritization.
var categoryWeight = map[Category]float64{
	CatSQLi:      1.00,
	CatIDOR:      1.00,
	CatXSS:       0.88,
	CatLFI:       0.85,
	CatSSRF:      0.82,
	CatCmdInject: 0.80,
	CatRedirect:  0.55,
	CatNoise:     0.30,
}

func classifyParam(name string, matched string) Category {
	lower := strings.ToLower(name)
	if matched == "weak" {
		return CatNoise
	}

	if _, _, ok := matchExclude(lower); ok {
		return CatNoise
	}

	sqliKeys := []string{
		"id", "search_term", "filter_by", "user_id", "product_id", "item_id",
		"order_id", "category_id", "article_id", "page_id", "catid", "cat_id",
		"pid", "uid", "nid", "gid", "tid", "select", "where", "table", "column",
	}
	for _, k := range sqliKeys {
		if lower == k || strings.Contains(lower, k) {
			return CatSQLi
		}
	}

	idorHints := []string{"_id", "_num", "_no", "_code", "record", "account", "invoice", "ref"}
	for _, h := range idorHints {
		if strings.Contains(lower, h) {
			return CatIDOR
		}
	}

	xssHints := []string{"q", "query", "search", "term", "keyword", "text", "msg", "message", "comment", "title", "name"}
	for _, h := range xssHints {
		if lower == h || strings.HasSuffix(lower, h) || strings.HasPrefix(lower, h+"_") {
			return CatXSS
		}
	}

	lfiHints := []string{"file", "path", "dir", "doc", "document", "template", "include", "page", "folder", "download"}
	for _, h := range lfiHints {
		if strings.Contains(lower, h) {
			return CatLFI
		}
	}

	ssrfHints := []string{"url", "uri", "link", "fetch", "proxy", "dest", "target", "host", "endpoint", "callback"}
	for _, h := range ssrfHints {
		if strings.Contains(lower, h) {
			return CatSSRF
		}
	}

	cmdHints := []string{"cmd", "exec", "command", "run", "shell", "process", "ping", "system"}
	for _, h := range cmdHints {
		if strings.Contains(lower, h) {
			return CatCmdInject
		}
	}

	redirectHints := []string{"redirect", "return", "next", "continue", "goto", "out", "redir"}
	for _, h := range redirectHints {
		if strings.Contains(lower, h) {
			return CatRedirect
		}
	}

	switch matched {
	case "high_exact", "high_suffix", "seclists", "pattern", "unknown":
		return CatSQLi
	case "weak":
		return CatNoise
	default:
		return CatNoise
	}
}

func weightedScore(base int, cat Category) int {
	w, ok := categoryWeight[cat]
	if !ok {
		w = 0.5
	}
	score := int(float64(base) * w)
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
