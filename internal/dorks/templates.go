package dorks

import "fmt"

// Template families for SQLi clone discovery.
const (
	FamilyPhraseClone  = "phrase_clone"
	FamilyKeywordMatch = "keyword_match"
	FamilyFiletype     = "filetype_stack"
	FamilyPathContext  = "path_context"
	FamilyPathLayout   = "path_layout"
	FamilyParamSurface = "param_surface"
	FamilySQLiError    = "sqli_error"
	FamilySQLiEndpoint = "sqli_endpoint"
	FamilySQLiDynamic  = "sqli_dynamic"
)

// TemplateCount is the number of base dork patterns across all families.
func TemplateCount() int {
	return len(phraseCloneTemplates()) +
		len(keywordMatchTemplates()) +
		len(filetypeTemplates()) +
		len(pathContextTemplates()) +
		len(pathLayoutTemplates()) +
		len(paramSurfaceTemplates()) +
		len(sqliErrorTemplates()) +
		len(sqliEndpointTemplates()) +
		len(sqliDynamicTemplates())
}

func phraseCloneTemplates() []func(phrase, param string) string {
	return []func(string, string) string{
		func(ph, pm string) string { return fmt.Sprintf(`intext:"%s" inurl:%s=`, ph, pm) },
		func(ph, pm string) string { return fmt.Sprintf(`inurl:%s= intext:"%s"`, pm, ph) },
		func(ph, pm string) string { return fmt.Sprintf(`intext:"%s" inurl:"%s="`, ph, pm) },
		func(ph, pm string) string { return fmt.Sprintf(`inurl:"%s=" "%s"`, pm, ph) },
		func(ph, pm string) string { return fmt.Sprintf(`intext:"%s" filetype:php inurl:%s=`, ph, pm) },
		func(ph, pm string) string { return fmt.Sprintf(`intext:"%s" filetype:asp inurl:%s=`, ph, pm) },
		func(ph, pm string) string { return fmt.Sprintf(`intext:"%s" inurl:%s= filetype:php`, ph, pm) },
		func(ph, pm string) string { return fmt.Sprintf(`allintext:%s inurl:%s=`, ph, pm) },
	}
}

func keywordMatchTemplates() []func(keyword, param string) string {
	return []func(string, string) string{
		func(kw, pm string) string { return fmt.Sprintf(`inurl:%s= intext:%s`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:%s= intext:"%s"`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:"%s=" "%s"`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`intext:"%s" inurl:"%s="`, kw, pm) },
		func(kw, pm string) string { return fmt.Sprintf(`intext:%s inurl:%s=`, kw, pm) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:%s= "%s"`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`allinurl:%s %s`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:%s intext:%s`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`"%s" inurl:%s=`, kw, pm) },
	}
}

func filetypeTemplates() []func(keyword, param string) string {
	return []func(string, string) string{
		func(kw, pm string) string { return fmt.Sprintf(`inurl:%s= intext:%s filetype:php`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:%s= intext:%s filetype:asp`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:%s= intext:%s filetype:jsp`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:%s= intext:%s filetype:aspx`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`filetype:php inurl:%s= intext:%s`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`filetype:asp inurl:%s= intext:%s`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`filetype:php inurl:%s=`, pm) },
		func(kw, pm string) string { return fmt.Sprintf(`filetype:asp inurl:%s=`, pm) },
	}
}

func pathContextTemplates() []func(keyword, param string) string {
	ctx := []string{"admin", "login", "search", "product", "view", "item", "detail", "list", "page", "catalog"}
	out := make([]func(string, string) string, 0, len(ctx))
	for _, c := range ctx {
		ctxPath := c
		out = append(out, func(kw, pm string) string {
			return fmt.Sprintf(`inurl:%s inurl:%s= intext:%s`, ctxPath, pm, kw)
		})
	}
	return out
}

func pathLayoutTemplates() []func(path, param string) string {
	return []func(string, string) string{
		func(path, pm string) string { return fmt.Sprintf(`inurl:%s inurl:%s=`, path, pm) },
		func(path, pm string) string { return fmt.Sprintf(`allinurl:%s %s`, path, pm) },
		func(path, pm string) string { return fmt.Sprintf(`inurl:%s filetype:php inurl:%s=`, path, pm) },
		func(path, pm string) string { return fmt.Sprintf(`inurl:%s filetype:asp inurl:%s=`, path, pm) },
		func(path, pm string) string { return fmt.Sprintf(`inurl:%s.php inurl:%s=`, path, pm) },
		func(path, pm string) string { return fmt.Sprintf(`inurl:%s/ inurl:%s=`, path, pm) },
	}
}

func paramSurfaceTemplates() []func(param string) string {
	return []func(string) string{
		func(pm string) string { return fmt.Sprintf(`inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:"%s="`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:%s filetype:php`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:%s filetype:asp`, pm) },
		func(pm string) string { return fmt.Sprintf(`filetype:php inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`filetype:asp inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:admin inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`inurl:view.php inurl:%s=`, pm) },
	}
}

// sqliErrorTemplates target pages already leaking database error messages.
func sqliErrorTemplates() []func(param string) string {
	return []func(string) string{
		func(pm string) string { return fmt.Sprintf(`intext:"You have an error in your SQL syntax" inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`intext:"mysql_fetch" inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`intext:"Warning: mysql" inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`intext:"Unclosed quotation mark" inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`intext:"ORA-01756" inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`intext:"Microsoft OLE DB Provider" inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`intext:"PostgreSQL query failed" inurl:%s=`, pm) },
		func(pm string) string { return fmt.Sprintf(`intext:"supplied argument is not a valid MySQL" inurl:%s=`, pm) },
	}
}

// sqliEndpointTemplates pair classic dynamic scripts with injectable parameters.
func sqliEndpointTemplates() []func(param string) string {
	scripts := []string{"view", "product", "news", "show", "detail", "article", "index", "page", "item", "catalog"}
	out := make([]func(string) string, 0, len(scripts))
	for _, script := range scripts {
		s := script
		out = append(out, func(pm string) string {
			return fmt.Sprintf(`inurl:%s.php inurl:%s=`, s, pm)
		})
	}
	return out
}

// sqliDynamicTemplates surface PHP/ASP query-string endpoints with theme keywords.
func sqliDynamicTemplates() []func(keyword, param string) string {
	return []func(string, string) string{
		func(kw, pm string) string { return fmt.Sprintf(`inurl:.php? inurl:%s= intext:%s`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:.asp? inurl:%s= intext:%s`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`ext:php inurl:? inurl:%s= intext:%s`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:.php?%s= intext:%s`, pm, kw) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:.php?%s=`, pm) },
		func(kw, pm string) string { return fmt.Sprintf(`inurl:php?%s= intext:%s`, pm, kw) },
	}
}

func applyPhraseClone(phrase, param string) []string {
	return applyPair(phraseCloneTemplates(), phrase, param)
}

func applyKeywordMatch(keyword, param string) []string {
	a := applyPair(keywordMatchTemplates(), keyword, param)
	b := applyPair(filetypeTemplates(), keyword, param)
	c := applyPair(pathContextTemplates(), keyword, param)
	return append(append(a, b...), c...)
}

func applyPathLayout(path, param string) []string {
	return applyPair(pathLayoutTemplates(), path, param)
}

func applyParamSurface(param string) []string {
	out := make([]string, 0, len(paramSurfaceTemplates())+len(sqliErrorTemplates())+len(sqliEndpointTemplates()))
	for _, fn := range paramSurfaceTemplates() {
		out = append(out, fn(param))
	}
	for _, fn := range sqliErrorTemplates() {
		out = append(out, fn(param))
	}
	for _, fn := range sqliEndpointTemplates() {
		out = append(out, fn(param))
	}
	return out
}

func applySQLiDynamic(keyword, param string) []string {
	return applyPair(sqliDynamicTemplates(), keyword, param)
}

func applyPair(fns []func(string, string) string, a, b string) []string {
	out := make([]string, 0, len(fns))
	for _, fn := range fns {
		out = append(out, fn(a, b))
	}
	return out
}

// buildDorks is kept for legacy tests.
func buildDorks(keyword, param string) []string {
	return applyKeywordMatch(keyword, param)
}
