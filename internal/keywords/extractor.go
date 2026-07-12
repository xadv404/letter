package keywords

import (
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

var tagWeights = map[string]int{
	"h1": 5, "h2": 4, "h3": 3,
	"h4": 2, "h5": 2, "h6": 2,
	"p": 1, "li": 1, "span": 1,
}

type Extractor struct {
	stopwords map[string]struct{}
	technical map[string]struct{}
	maxCount  int
	seen      map[string]int
}

func New(maxCount int) *Extractor {
	return &Extractor{
		stopwords: buildStopwords(),
		technical: buildTechnical(),
		maxCount:  maxCount,
		seen:      map[string]int{},
	}
}

func (e *Extractor) Extract(domain string, doc *html.Node) []Result {
	var results []Result
	var walk func(*html.Node, int)
	walk = func(n *html.Node, weight int) {
		if n.Type == html.ElementNode {
			if w, ok := tagWeights[n.Data]; ok {
				weight = w
			}
			if weight > 0 && n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				text := clean(n.FirstChild.Data)
				for _, token := range tokenize(text) {
					if e.accept(token) {
						e.seen[token] += weight
						results = append(results, Result{Domain: domain, Keyword: token, Weight: weight})
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, weight)
		}
	}
	walk(doc, 0)

	// multi-word pairs from accumulated tokens
	tokens := e.topTokens(50)
	for i := 0; i < len(tokens); i++ {
		for j := i + 1; j < len(tokens) && j < i+4; j++ {
			pair := tokens[i] + " " + tokens[j]
			if len(pair) >= 2 && len(pair) <= 100 && e.acceptPair(tokens[i], tokens[j]) {
				results = append(results, Result{Domain: domain, Keyword: pair, Weight: 2})
			}
		}
	}

	if len(results) > e.maxCount {
		results = results[:e.maxCount]
	}
	return results
}

func (e *Extractor) Count() int { return len(e.seen) }

type Result struct {
	Domain  string
	Keyword string
	Weight  int
}

func (e *Extractor) accept(word string) bool {
	if len(word) < 2 || len(word) > 100 {
		return false
	}
	if _, ok := e.stopwords[word]; ok {
		return false
	}
	if _, ok := e.technical[word]; ok {
		return false
	}
	if _, seen := e.seen[word]; seen && e.seen[word] > 0 {
		return false
	}
	return true
}

func (e *Extractor) acceptPair(a, b string) bool {
	return e.accept(a) || e.accept(b)
}

func (e *Extractor) topTokens(limit int) []string {
	type kv struct {
		k string
		v int
	}
	arr := make([]kv, 0, len(e.seen))
	for k, v := range e.seen {
		arr = append(arr, kv{k, v})
	}
	for i := 0; i < len(arr); i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[j].v > arr[i].v {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
	if len(arr) > limit {
		arr = arr[:limit]
	}
	out := make([]string, len(arr))
	for i, item := range arr {
		out[i] = item.k
	}
	return out
}

func clean(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return r
		}
		return ' '
	}, s)
}

func tokenize(s string) []string {
	parts := strings.Fields(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func buildStopwords() map[string]struct{} {
	words := []string{
		"a", "an", "the", "and", "or", "but", "if", "then", "else", "when", "at", "by", "for", "with",
		"about", "against", "between", "into", "through", "during", "before", "after", "above", "below",
		"to", "from", "up", "down", "in", "out", "on", "off", "over", "under", "again", "further",
		"is", "am", "are", "was", "were", "be", "been", "being", "have", "has", "had", "do", "does",
		"did", "will", "would", "shall", "should", "can", "could", "may", "might", "must", "not",
		"this", "that", "these", "those", "i", "you", "he", "she", "it", "we", "they", "me", "him",
		"her", "us", "them", "my", "your", "his", "its", "our", "their", "what", "which", "who",
		"whom", "whose", "where", "why", "how", "all", "each", "every", "both", "few", "more",
		"most", "other", "some", "such", "no", "nor", "only", "own", "same", "so", "than", "too",
		"very", "just", "also", "now", "here", "there", "once", "get", "got", "go", "going", "gone",
		"see", "seen", "use", "used", "using", "make", "made", "new", "old", "one", "two", "three",
		"click", "read", "more", "home", "page", "menu", "search", "login", "logout", "sign", "copyright",
		"privacy", "policy", "terms", "contact", "email", "phone", "address", "welcome", "learn", "today",
	}
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}

func buildTechnical() map[string]struct{} {
	words := []string{
		"html", "css", "javascript", "json", "xml", "http", "https", "www", "div", "span", "class",
		"script", "style", "meta", "charset", "utf", "viewport", "jquery", "bootstrap", "webpack",
		"react", "angular", "vue", "php", "asp", "jsp", "cookie", "session", "token", "csrf",
	}
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}
