package keywords

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

var pageExtSuffix = regexp.MustCompile(`(?i)\.(php|asp|aspx|jsp|cfm|html?)$`)

var tagWeights = map[string]int{
	"title": 5,
	"h1": 5, "h2": 4, "h3": 3,
	"h4": 2, "h5": 2, "h6": 2,
	"a": 2,
	"p": 1, "li": 1, "span": 1,
}

// Result is a scored keyword discovered during extraction.
type Result struct {
	Domain  string
	Keyword string
	Weight  int
	Source  string // token, bigram, trigram, heading
}

type block struct {
	tag    string
	weight int
	text   string
}

// Extractor performs weighted, context-aware keyword extraction across a crawl session.
type Extractor struct {
	filter     *Filter
	maxSession int

	scores        map[string]int
	domainScores  map[string]map[string]int
	emitted       map[string]int
	sessionLen    int
}

func New(maxCount int) *Extractor {
	return &Extractor{
		filter:     NewFilter(),
		maxSession: maxCount,
		scores:       map[string]int{},
		domainScores: map[string]map[string]int{},
		emitted:      map[string]int{},
	}
}

func (e *Extractor) Extract(domain string, doc *html.Node) []Result {
	if e.sessionLen >= e.maxSession {
		return nil
	}

	blocks := collectBlocks(doc)
	candidates := map[string]candidate{}

	for _, blk := range blocks {
		e.processBlock(blk, candidates)
	}

	return e.commitCandidates(domain, candidates)
}

// ExtractFromURL scores keywords from a crawled URL path (requires the page to be visited).
func (e *Extractor) ExtractFromURL(domain, rawURL string) []Result {
	if e.sessionLen >= e.maxSession {
		return nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	candidates := map[string]candidate{}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return nil
	}

	for _, seg := range splitPathSegments(path) {
		seg = pageExtSuffix.ReplaceAllString(seg, "")
		if seg == "" || !e.filter.AcceptToken(seg) {
			continue
		}
		mergeCandidate(candidates, seg, 4, "url-path")
	}

	return e.commitCandidates(domain, candidates)
}

func splitPathSegments(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '_' || r == '-' || r == '.'
	})
}

func (e *Extractor) commitCandidates(domain string, candidates map[string]candidate) []Result {
	var results []Result
	for phrase, cand := range candidates {
		if !e.filter.AcceptPhrase(phrase) && !e.filter.AcceptToken(phrase) {
			continue
		}
		e.scores[phrase] += cand.weight
		newScore := e.scores[phrase]
		e.addDomainScore(domain, phrase, cand.weight)
		if newScore <= e.emitted[phrase] {
			continue
		}
		if e.sessionLen >= e.maxSession {
			break
		}
		e.emitted[phrase] = newScore
		e.sessionLen++
		results = append(results, Result{
			Domain:  domain,
			Keyword: phrase,
			Weight:  newScore,
			Source:  cand.source,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Weight == results[j].Weight {
			return results[i].Keyword < results[j].Keyword
		}
		return results[i].Weight > results[j].Weight
	})
	return results
}

func (e *Extractor) Count() int      { return e.sessionLen }
func (e *Extractor) Unique() int     { return len(e.scores) }
func (e *Extractor) Stopwords() int  { return len(e.filter.stopwords) }

type candidate struct {
	weight int
	source string
}

func (e *Extractor) processBlock(blk block, out map[string]candidate) {
	text := cleanText(blk.text)
	if text == "" {
		return
	}

	tokens := tokenize(text)
	valid := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if e.filter.AcceptToken(t) {
			valid = append(valid, t)
		}
	}

	// Full heading phrase — high value for recon.
	if isHeading(blk.tag) && e.filter.AcceptPhrase(text) {
		weight := blk.weight * 2
		mergeCandidate(out, text, weight, "heading")
	}

	// Single tokens inherit block weight.
	for _, tok := range valid {
		mergeCandidate(out, tok, blk.weight, "token")
	}

	// Adjacent bigrams/trigrams from the same block preserve context.
	for i := 0; i < len(valid); i++ {
		if i+1 < len(valid) {
			bigram := valid[i] + " " + valid[i+1]
			w := pairWeight(blk.weight, 2)
			mergeCandidate(out, bigram, w, "bigram")
		}
		if i+2 < len(valid) && blk.weight >= 3 {
			trigram := valid[i] + " " + valid[i+1] + " " + valid[i+2]
			w := pairWeight(blk.weight, 3)
			mergeCandidate(out, trigram, w, "trigram")
		}
	}

	// Cross-token pairs inside high-weight blocks (headings).
	if blk.weight >= 3 && len(valid) >= 2 {
		limit := min(len(valid), 8)
		for i := 0; i < limit; i++ {
			for j := i + 1; j < limit && j < i+3; j++ {
				pair := valid[i] + " " + valid[j]
				if strings.Contains(pair, " ") {
					mergeCandidate(out, pair, pairWeight(blk.weight, 2), "pair")
				}
			}
		}
	}
}

func collectBlocks(doc *html.Node) []block {
	var blocks []block
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if w, ok := tagWeights[n.Data]; ok {
				text := deepText(n)
				if strings.TrimSpace(text) != "" {
					blocks = append(blocks, block{tag: n.Data, weight: w, text: text})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return blocks
}

func deepText(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && isSkippedTag(c.Data) {
			continue
		}
		b.WriteString(deepText(c))
		if c.Type == html.ElementNode && needsSpace(c.Data) {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func isSkippedTag(tag string) bool {
	switch tag {
	case "script", "style", "noscript", "svg", "path", "code", "pre":
		return true
	default:
		return false
	}
}

func needsSpace(tag string) bool {
	switch tag {
	case "p", "li", "br", "div", "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	default:
		return false
	}
}

func isHeading(tag string) bool {
	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	default:
		return false
	}
}

func mergeCandidate(out map[string]candidate, phrase string, weight int, source string) {
	phrase = strings.Join(strings.Fields(strings.TrimSpace(phrase)), " ")
	if phrase == "" || len(phrase) < MinKeywordLen || len(phrase) > MaxKeywordLen {
		return
	}
	cur, ok := out[phrase]
	if !ok || weight > cur.weight {
		out[phrase] = candidate{weight: weight, source: source}
	}
}

func pairWeight(blockWeight, n int) int {
	if n <= 1 {
		return blockWeight
	}
	w := blockWeight - (n - 1)
	if w < 1 {
		return 1
	}
	return w
}

func cleanText(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ' {
			return r
		}
		return ' '
	}, s)
}

func tokenize(s string) []string {
	return strings.Fields(cleanText(s))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (e *Extractor) addDomainScore(domain, phrase string, weight int) {
	if e.domainScores[domain] == nil {
		e.domainScores[domain] = map[string]int{}
	}
	e.domainScores[domain][phrase] += weight
}

// Top returns the highest-scoring keywords accumulated in the session.
func (e *Extractor) Top(limit int) []Result {
	return e.topFrom(e.scores, limit)
}

// TopForDomain returns top keywords for a specific domain.
func (e *Extractor) TopForDomain(domain string, limit int) []Result {
	if e.domainScores[domain] == nil {
		return nil
	}
	results := e.topFrom(e.domainScores[domain], limit)
	for i := range results {
		results[i].Domain = domain
	}
	return results
}

func (e *Extractor) topFrom(scores map[string]int, limit int) []Result {
	type kv struct {
		k string
		v int
	}
	arr := make([]kv, 0, len(scores))
	for k, v := range scores {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].v == arr[j].v {
			return arr[i].k < arr[j].k
		}
		return arr[i].v > arr[j].v
	})
	if limit > 0 && len(arr) > limit {
		arr = arr[:limit]
	}
	out := make([]Result, len(arr))
	for i, item := range arr {
		out[i] = Result{Keyword: item.k, Weight: item.v}
	}
	return out
}
