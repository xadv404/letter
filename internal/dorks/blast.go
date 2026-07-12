package dorks

import "fmt"

// burpTopInjectable — high-hit parameter names from SecLists / bug bounty SQLi hunts.
var burpTopInjectable = []string{
	"id", "pid", "cid", "nid", "gid", "tid", "uid", "sid", "oid", "rid", "lid", "aid", "vid", "mid", "did",
	"cat", "catid", "cat_id", "category", "categoryid", "category_id", "cat_id",
	"product_id", "productid", "item_id", "itemid", "article_id", "articleid", "page_id", "pageid",
	"user_id", "userid", "order_id", "orderid", "news_id", "newsid", "forum_id", "thread_id", "post_id",
	"comment_id", "photo_id", "album_id", "image_id", "video_id", "file_id", "doc_id", "download_id",
	"listing_id", "record_id", "object_id", "parent_id", "group_id", "event_id", "invoice_id",
	"search_term", "searchterm", "filter_by", "filterby", "query", "q", "keyword", "search", "filter",
	"num", "idx", "ref", "code", "no", "nr", "select", "table", "column", "field", "row", "record",
	"view", "show", "detail", "item", "product", "article", "news", "page", "menu_id", "module_id",
}

// coreInjectable alias for tests.
var coreInjectable = burpTopInjectable

// globalLiteralBlast are fixed ultra-high-yield dorks (1 operator, max Google results).
func globalLiteralBlast() []string {
	hits := []string{"id", "cat", "pid", "cid", "nid", "gid", "uid", "sid", "product_id", "item_id", "article_id", "page_id", "user_id", "category_id", "news_id", "search_term"}
	scripts := []string{"index", "view", "product", "news", "article", "detail", "show", "page", "item", "catalog", "list", "category", "shop", "search", "profile", "download", "forum", "gallery", "content", "read"}
	exts := []string{"php", "asp", "aspx", "jsp"}

	var out []string
	for _, pm := range hits {
		out = append(out,
			fmt.Sprintf(`inurl:.php?%s=`, pm),
			fmt.Sprintf(`inurl:.asp?%s=`, pm),
			fmt.Sprintf(`inurl:?%s=`, pm),
			fmt.Sprintf(`"%s="`, pm),
		)
	}
	for _, s := range scripts {
		for _, pm := range hits[:10] {
			for _, ext := range exts[:2] {
				out = append(out, fmt.Sprintf(`inurl:%s.%s?%s=`, s, ext, pm))
			}
		}
	}
	out = append(out,
		`inurl:.php?id=`,
		`inurl:.php?cat=`,
		`inurl:.php?pid=`,
		`inurl:.asp?id=`,
		`inurl:index.php?id=`,
		`inurl:view.php?id=`,
		`inurl:product.php?id=`,
		`inurl:news.php?id=`,
		`inurl:article.php?id=`,
		`inurl:detail.php?id=`,
		`inurl:show.php?id=`,
		`inurl:page.php?id=`,
		`inurl:category.php?cat=`,
		`inurl:product.php?cat=`,
		`inurl:view.php?cat=`,
		`inurl:shop.php?id=`,
		`inurl:catalog.php?id=`,
		`inurl:item.php?id=`,
		`inurl:list.php?id=`,
		`inurl:search.php?q=`,
		`inurl:profile.php?id=`,
		`inurl:download.php?id=`,
		`inurl:forum.php?id=`,
		`inurl:gallery.php?id=`,
		`inurl:content.php?id=`,
		`inurl:read.php?id=`,
		`inurl:display.php?id=`,
		`inurl:print.php?id=`,
		`inurl:report.php?id=`,
		`inurl:order.php?id=`,
		`inurl:cart.php?id=`,
		`inurl:checkout.php?id=`,
		`inurl:member.php?id=`,
		`inurl:user.php?id=`,
		`inurl:admin.php?id=`,
		`inurl:login.php?id=`,
	)
	return out
}

// ultraSimpleTemplates — single-operator dorks (5-10× more Google results than multi-inurl).
func ultraSimpleTemplates() []func(param string) string {
	return []func(string) string{
		func(pm string) string { return fmt.Sprintf(`inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:"%s="`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`"%s="`, pm) },
		func(pm string) string { return fmt.Sprintf(`"?%s="`, pm) },
		func(pm string) string { return fmt.Sprintf(`allinurl:%s`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:%s&`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:&%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.php?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.asp?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.aspx?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.jsp?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.cfm?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.php?%s=1`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.asp?%s=1`, pm) },
	}
}

// literalEndpointTemplates — single inurl with script?param= (much higher yield than double inurl).
func literalEndpointTemplates() []func(script, param string) string {
	exts := []string{"php", "asp", "aspx"}
	var fns []func(string, string) string
	for _, ext := range exts {
		e := ext
		fns = append(fns, func(script, pm string) string {
			return fmt.Sprintf(`inurl:%s.%s?%s=`, script, e, pm)
		})
	}
	return fns
}

func applyUltraSimple(param string) []string {
	return applyParamOnly(ultraSimpleTemplates(), param)
}

func applyLiteralEndpoints(param string) []string {
	var out []string
	for _, script := range classicScripts {
		for _, fn := range literalEndpointTemplates() {
			out = append(out, fn(script, param))
		}
	}
	return out
}

func applyParamOnly(fns []func(string) string, param string) []string {
	out := make([]string, 0, len(fns))
	for _, fn := range fns {
		out = append(out, fn(param))
	}
	return out
}

// splitParamTiers separates high-confidence injectable params from the rest.
func splitParamTiers(params []string) (primary, all []string) {
	all = append([]string{}, params...)
	seen := map[string]struct{}{}
	for _, p := range params {
		if isPrimaryInjectable(p) {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				primary = append(primary, p)
			}
		}
	}
	if len(primary) == 0 {
		primary = all
	}
	if len(primary) > 60 {
		primary = primary[:60]
	}
	return primary, all
}

func isPrimaryInjectable(name string) bool {
	if IsInjectableParam(name, 65) {
		return true
	}
	for _, core := range burpTopInjectable[:40] {
		if name == core {
			return true
		}
	}
	return false
}
