package dorks

import "fmt"

// classicScripts are dynamic endpoints commonly exposing injectable query params.
var classicScripts = []string{
	"index", "view", "product", "products", "news", "show", "detail", "article", "page",
	"item", "catalog", "list", "display", "read", "gallery", "category", "shop", "cart",
	"search", "results", "profile", "user", "member", "download", "redirect", "content",
	"event", "events", "photo", "image", "video", "forum", "thread", "post", "comment",
	"admin", "login", "register", "order", "checkout", "invoice", "report", "print",
}

// pathContexts are URL segments that correlate with database-backed pages.
var pathContexts = []string{
	"admin", "login", "search", "product", "products", "view", "item", "detail",
	"list", "page", "catalog", "shop", "category", "article", "news", "member",
	"profile", "cart", "checkout", "forum", "gallery", "download",
}

var stackExtensions = []string{"php", "asp", "aspx", "jsp", "cfm"}

// volumeParamTemplates are high-yield dorks without intext:keyword (max Google results).
func volumeParamTemplates() []func(param string) string {
	out := []func(string) string{
		func(pm string) string { return fmt.Sprintf(`inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:"%s="`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:%s&`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:&%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.php?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.asp?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.aspx?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.jsp?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.cfm?%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.php? inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:.asp? inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`ext:php inurl:? inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`ext:asp inurl:? inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`ext:aspx inurl:? inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`allinurl:%s =`, pm) },
	}
	for _, ext := range stackExtensions {
		e := ext
		out = append(out, func(pm string) string {
			return fmt.Sprintf(`filetype:%s inurl:%s=`, e, pm)
		})
	}
	for _, ctx := range pathContexts {
		c := ctx
		out = append(out, func(pm string) string {
			return fmt.Sprintf(`inurl:%s inurl:%s=`, c, pm)
		})
	}
	for _, script := range classicScripts {
		s := script
		out = append(out, func(pm string) string {
			return fmt.Sprintf(`inurl:%s.php inurl:%s=`, s, pm)
		})
		out = append(out, func(pm string) string {
			return fmt.Sprintf(`inurl:%s.asp inurl:%s=`, s, pm)
		})
	}
	return out
}

func applyVolumeParam(param string) []string {
	fns := volumeParamTemplates()
	out := make([]string, 0, len(fns))
	for _, fn := range fns {
		out = append(out, fn(param))
	}
	return out
}

func applyVolumePath(path, param string) []string {
	return []string{
		fmt.Sprintf(`inurl:%s inurl:%s=`, path, param),
		fmt.Sprintf(`inurl:%s.php inurl:%s=`, path, param),
		fmt.Sprintf(`inurl:%s.asp inurl:%s=`, path, param),
		fmt.Sprintf(`allinurl:%s %s`, path, param),
		fmt.Sprintf(`inurl:%s/ inurl:%s=`, path, param),
		fmt.Sprintf(`inurl:%s filetype:php inurl:%s=`, path, param),
		fmt.Sprintf(`inurl:%s filetype:asp inurl:%s=`, path, param),
		fmt.Sprintf(`inurl:%s?%s=`, path, param),
	}
}

func applyMultiParam(params []string) []string {
	var out []string
	n := len(params)
	if n < 2 {
		return nil
	}
	if n > 25 {
		n = 25
		params = params[:n]
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n && j < i+6; j++ {
			a, b := params[i], params[j]
			out = append(out,
				fmt.Sprintf(`inurl:%s= inurl:%s=`, a, b),
				fmt.Sprintf(`allinurl:%s %s`, a, b),
				fmt.Sprintf(`inurl:.php? inurl:%s= inurl:%s=`, a, b),
				fmt.Sprintf(`inurl:%s= inurl:%s&`, a, b),
			)
		}
	}
	return out
}
