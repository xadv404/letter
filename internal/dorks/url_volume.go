package dorks

import (
	"fmt"
	"strings"
)

// MaxURLVolumeDorks — few dorks, each tuned for millions of Google results.
const MaxURLVolumeDorks = 50

// fixedVolumeDorks are hand-picked single-operator patterns (max URL breadth).
var fixedVolumeDorks = []string{
	// Param seul — millions de résultats chacun
	"inurl:id=",
	"inurl:cat=",
	"inurl:pid=",
	"inurl:cid=",
	"inurl:uid=",
	"inurl:nid=",
	"inurl:gid=",
	"inurl:sid=",
	"inurl:product_id=",
	"inurl:item_id=",
	"inurl:article_id=",
	"inurl:user_id=",
	// PHP large
	"inurl:.php?id=",
	"inurl:.php?cat=",
	"inurl:.php?pid=",
	"inurl:.php?product_id=",
	"inurl:.php?item_id=",
	"inurl:.php?article_id=",
	"inurl:.php?user_id=",
	"inurl:.php?cat_id=",
	// ASP
	"inurl:.asp?id=",
	"inurl:.asp?cat=",
	"inurl:.asp?pid=",
	// Query-string
	"inurl:?id=",
	"inurl:?cat=",
	"inurl:?pid=",
	// Scripts littéraux — très large
	"inurl:view.php?id=",
	"inurl:product.php?id=",
	"inurl:product.php?cat=",
	"inurl:news.php?id=",
	"inurl:index.php?id=",
	"inurl:article.php?id=",
	"inurl:detail.php?id=",
	"inurl:show.php?id=",
	"inurl:shop.php?id=",
	"inurl:catalog.php?id=",
	"inurl:page.php?id=",
	"inurl:item.php?id=",
	"inurl:list.php?id=",
	"inurl:category.php?cat=",
	"inurl:forum.php?id=",
	"inurl:download.php?id=",
	"inurl:profile.php?id=",
	"inurl:content.php?id=",
	"inurl:gallery.php?id=",
	"inurl:search.php?id=",
}

// DorkSet holds the final dork output (volume-URL strategy).
type DorkSet struct {
	Exploitable []string // same as Volume — kept for API compat
	Volume      []string
	All         []string
}

// GenerateURLVolume returns ~50 ultra-broad dorks targeting 200-500k URLs total.
func GenerateURLVolume(crawledParams []string) DorkSet {
	seen := map[string]struct{}{}
	var out []string

	add := func(d string) {
		d = strings.TrimSpace(d)
		if d == "" || len(out) >= MaxURLVolumeDorks {
			return
		}
		if _, ok := seen[d]; ok {
			return
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}

	for _, d := range fixedVolumeDorks {
		add(d)
	}

	// Inject up to 8 crawled params (broad form only — no intext, no double inurl).
	for _, pm := range volumeCrawledParams(crawledParams) {
		if len(out) >= MaxURLVolumeDorks {
			break
		}
		add(fmt.Sprintf("inurl:%s=", pm))
		if len(out) >= MaxURLVolumeDorks {
			break
		}
		add(fmt.Sprintf("inurl:.php?%s=", pm))
	}

	return DorkSet{Exploitable: out, Volume: out, All: out}
}

// volumeCrawledParams picks broad SQLi params from crawl not already in fixed set.
func volumeCrawledParams(crawled []string) []string {
	fixedParams := map[string]struct{}{
		"id": {}, "cat": {}, "pid": {}, "cid": {}, "uid": {}, "nid": {}, "gid": {}, "sid": {},
		"product_id": {}, "item_id": {}, "article_id": {}, "user_id": {}, "cat_id": {},
	}
	var out []string
	seen := map[string]struct{}{}
	for _, p := range crawled {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" || len(p) > 32 {
			continue
		}
		if _, inFixed := fixedParams[p]; inFixed {
			continue
		}
		if !isVolumeParam(p) {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
		if len(out) >= 4 {
			break
		}
	}
	return out
}

func isVolumeParam(name string) bool {
	if isEliteParam(name) {
		return true
	}
	return strings.HasSuffix(name, "_id") && !strings.Contains(name, "page")
}
