package dorks

import "fmt"

// Template families (50 patterns total) for SQLi clone discovery.
const (
	FamilyPhraseClone  = "phrase_clone"
	FamilyKeywordMatch = "keyword_match"
	FamilyFiletype     = "filetype_stack"
	FamilyPathContext  = "path_context"
	FamilyPathLayout   = "path_layout"
	FamilyParamSurface = "param_surface"
)

// TemplateCount is the number of base dork patterns (50 across 6 families).
func TemplateCount() int {
	return len(phraseCloneTemplates()) +
		len(keywordMatchTemplates()) +
		len(filetypeTemplates()) +
		len(pathContextTemplates()) +
		len(pathLayoutTemplates()) +
		len(paramSurfaceTemplates())
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
		func(kw, pm string) string { return fmt.Sprintf(`inurl:%s= | inurl:%s intext:%s`, pm, pm, kw) },
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
	out := make([]string, 0, len(paramSurfaceTemplates()))
	for _, fn := range paramSurfaceTemplates() {
		out = append(out, fn(param))
	}
	return out
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
