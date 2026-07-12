package dorks

import "fmt"

// classicScripts are dynamic endpoints commonly exposing injectable query params.
var classicScripts = []string{
	"index", "view", "product", "products", "news", "show", "detail", "article", "page",
	"item", "catalog", "list", "display", "read", "gallery", "category", "shop", "cart",
	"search", "results", "profile", "user", "member", "download", "redirect", "content",
	"event", "events", "photo", "image", "video", "forum", "thread", "post", "comment",
	"admin", "login", "register", "order", "checkout", "invoice", "report", "print",
	"shop", "buy", "store", "browse", "info", "main", "home", "default", "file", "get",
}

// stackExtensions for filetype/ext filters.
var stackExtensions = []string{"php", "asp", "aspx", "jsp", "cfm", "pl", "cgi"}

// applyInjectableErrors — error-leak dorks (pages already broken = injectable).
func applyInjectableErrors(param string) []string {
	templates := []string{
		`intext:"You have an error in your SQL syntax" inurl:%s=`,
		`intext:"mysql_fetch" inurl:%s=`,
		`intext:"Warning: mysql" inurl:%s=`,
		`intext:"Unclosed quotation mark" inurl:%s=`,
		`intext:"ORA-01756" inurl:%s=`,
		`intext:"Microsoft OLE DB Provider" inurl:%s=`,
		`intext:"PostgreSQL query failed" inurl:%s=`,
		`intext:"SQL syntax" inurl:%s=`,
		`intext:"mysql_num_rows" inurl:%s=`,
		`intext:"mysqli_" inurl:%s=`,
		`intext:"pg_query" inurl:%s=`,
		`intext:"sqlite_" inurl:%s=`,
	}
	out := make([]string, 0, len(templates))
	for _, t := range templates {
		out = append(out, fmt.Sprintf(t, param))
	}
	return out
}

// applyStackFiletype — param + stack (2 operators max, injectable focus).
func applyStackFiletype(param string) []string {
	var out []string
	for _, ext := range stackExtensions {
		out = append(out,
			fmt.Sprintf(`filetype:%s inurl:%s=`, ext, param),
			fmt.Sprintf(`ext:%s inurl:?%s=`, ext, param),
		)
	}
	return out
}

// applySimplePath — top paths only, 2 operators max.
func applySimplePath(path, param string) []string {
	return []string{
		fmt.Sprintf(`inurl:%s?%s=`, path, param),
		fmt.Sprintf(`inurl:%s.%s?%s=`, path, "php", param),
		fmt.Sprintf(`inurl:%s/ inurl:%s=`, path, param),
	}
}

func applyMultiParam(params []string) []string {
	var out []string
	n := len(params)
	if n < 2 {
		return nil
	}
	if n > 40 {
		n = 40
		params = params[:n]
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n && j < i+10; j++ {
			a, b := params[i], params[j]
			out = append(out,
				fmt.Sprintf(`inurl:%s= inurl:%s=`, a, b),
				fmt.Sprintf(`inurl:.php?%s= inurl:%s=`, a, b),
				fmt.Sprintf(`allinurl:%s %s`, a, b),
				fmt.Sprintf(`inurl:%s= inurl:%s&`, a, b),
				fmt.Sprintf(`"%s=" "%s="`, a, b),
			)
		}
	}
	return out
}

// volumeParamTemplates kept for TemplateCount; simplified secondary tier.
func volumeParamTemplates() []func(param string) string {
	return ultraSimpleTemplates()
}

func applyVolumeParam(param string) []string {
	return applyUltraSimple(param)
}

func applyVolumePath(path, param string) []string {
	return applySimplePath(path, param)
}
