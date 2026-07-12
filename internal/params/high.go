package params

import "strings"

// HIGH (≥85): classically injectable parameters.
var highExact = map[string]int{
	"id": 95, "search_term": 94, "filter_by": 93,
	"user_id": 91, "product_id": 91, "item_id": 90, "order_id": 90,
	"category_id": 89, "article_id": 89, "page_id": 88, "catid": 88, "cat_id": 88,
	"pid": 86, "uid": 86, "nid": 86, "gid": 85, "tid": 85,
}

var highSuffixes = []struct {
	suffix string
	score  int
}{
	{"_id", 86},
	{"_num", 85},
	{"_no", 85},
	{"_code", 85},
}

// LOW (50–64): less probable injection points (excluded by default threshold 65).
var lowExact = map[string]int{
	"page": 58, "sort": 56, "category": 58, "cat": 55, "order": 60,
	"limit": 58, "offset": 58, "start": 55, "end": 55,
	"a": 52, "b": 52, "c": 52, "p": 54, "n": 54, "x": 52, "y": 52,
	"val": 58, "value": 60, "data": 58, "obj": 55, "key": 58,
	"arg": 55, "param": 56, "input": 58, "output": 56,
	"no": 55, "nr": 55, "num": 62, "idx": 62,
}

// mediumPatterns: fuzzy SQLi-relevant fragments (65–84).
var mediumPatterns = []struct {
	contains string
	score    int
	reason   string
}{
	{"id", 74, "contains id fragment"},
	{"num", 72, "contains num fragment"},
	{"number", 73, "contains number fragment"},
	{"ref", 70, "contains ref fragment"},
	{"code", 71, "contains code fragment"},
	{"select", 76, "contains select fragment"},
	{"where", 76, "contains where fragment"},
	{"table", 75, "contains table fragment"},
	{"column", 75, "contains column fragment"},
	{"field", 72, "contains field fragment"},
	{"record", 72, "contains record fragment"},
	{"row", 70, "contains row fragment"},
	{"index", 70, "contains index fragment"},
	{"account", 73, "contains account fragment"},
	{"invoice", 72, "contains invoice fragment"},
	{"report", 71, "contains report fragment"},
	{"search", 72, "contains search fragment"},
}

func scoreHighExact(name string) (int, bool) {
	if s, ok := highExact[name]; ok {
		return s, true
	}
	return 0, false
}

func scoreHighSuffix(name string) (int, bool) {
	for _, suf := range highSuffixes {
		if len(name) > len(suf.suffix) && strings.HasSuffix(name, suf.suffix) {
			return suf.score, true
		}
	}
	return 0, false
}

func scoreMediumPattern(name string) (int, string, bool) {
	best := 0
	reason := ""
	for _, p := range mediumPatterns {
		if strings.Contains(name, p.contains) && p.score > best {
			best = p.score
			reason = p.reason
		}
	}
	if best > 0 {
		return best, reason, true
	}
	return 0, "", false
}

func scoreLowExact(name string) (int, bool) {
	if s, ok := lowExact[name]; ok {
		return s, true
	}
	if len(name) == 1 {
		return 52, true
	}
	return 0, false
}
