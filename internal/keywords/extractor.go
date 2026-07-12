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
	"th": 3, "label": 3, "button": 2,
	"a": 2,
	"td": 2, "dt": 2, "dd": 2,
	"p": 1, "li": 1, "span": 1,
}

// Result is a scored keyword discovered during extraction.
type Result struct {
	Domain  string
	Keyword string
	Weight  int
	Source  string // token, bigram, trigram, heading, meta, url-path
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

	scores         map[string]int
	domainScores    map[string]map[string]int
	domainPathScores map[string]map[string]int
	domainPageHits  map[string]map[string]int
	domainPageSeen map[string]map[string]map[string]struct{}
	emitted        map[string]int
	sessionLen     int
}

func New(maxCount int) *Extractor {
	return &Extractor{
		filter:         NewFilter(),
		maxSession:     maxCount,
		scores:         map[string]int{},
		domainScores:     map[string]map[string]int{},
		domainPathScores: map[string]map[string]int{},
		domainPageHits:   map[string]map[string]int{},
		domainPageSeen: map[string]map[string]map[string]struct{}{},
		emitted:        map[string]int{},
	}
}

// ExtractPage extracts keywords from a crawled URL and its HTML document.
func (e *Extractor) ExtractPage(domain, pageURL string, doc *html.Node) []Result {
	if e.sessionLen >= e.maxSession {
		return nil
	}

	candidates := map[string]candidate{}
	e.collectURLPath(pageURL, candidates)
	if doc != nil {
		for _, blk := range collectMeta(doc) {
			e.processBlock(blk, candidates)
		}
		for _, blk := range collectBlocks(doc) {
			e.processBlock(blk, candidates)
		}
	}
	return e.commitCandidates(domain, pageURL, candidates)
}

// Extract extracts keywords from HTML only (used in tests).
func (e *Extractor) Extract(domain string, doc *html.Node) []Result {
	return e.ExtractPage(domain, "", doc)
}

// ExtractFromURL extracts keywords from a URL path only (used in tests).
func (e *Extractor) ExtractFromURL(domain, rawURL string) []Result {
	return e.ExtractPage(domain, rawURL, nil)
}

func (e *Extractor) collectURLPath(rawURL string, out map[string]candidate) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return
	}

	for _, seg := range splitPathSegments(path) {
		seg = pageExtSuffix.ReplaceAllString(seg, "")
		for _, part := range expandCompoundToken(seg) {
			if e.filter.AcceptToken(part) {
				mergeCandidate(out, part, 5, "url-path")
			}
		}
	}
}

func splitPathSegments(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '_' || r == '-' || r == '.'
	})
}

func (e *Extractor) commitCandidates(domain, pageURL string, candidates map[string]candidate) []Result {
	var results []Result
	for phrase, cand := range candidates {
		if !e.filter.AcceptPhrase(phrase) && !e.filter.AcceptToken(phrase) {
			continue
		}
		weight := int(float64(cand.weight) * reconWeightMultiplier(phrase))
		if weight < 1 {
			weight = 1
		}

		e.scores[phrase] += weight
		newScore := e.scores[phrase]
		e.addDomainScore(domain, phrase, weight)
		if cand.source == "url-path" {
			e.addPathScore(domain, phrase, weight)
		}
		if pageURL != "" {
			e.recordPageHit(domain, pageURL, phrase)
		}
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

func (e *Extractor) recordPageHit(domain, pageURL, keyword string) {
	if e.domainPageSeen[domain] == nil {
		e.domainPageSeen[domain] = map[string]map[string]struct{}{}
	}
	if e.domainPageSeen[domain][keyword] == nil {
		e.domainPageSeen[domain][keyword] = map[string]struct{}{}
	}
	if _, seen := e.domainPageSeen[domain][keyword][pageURL]; seen {
		return
	}
	e.domainPageSeen[domain][keyword][pageURL] = struct{}{}
	if e.domainPageHits[domain] == nil {
		e.domainPageHits[domain] = map[string]int{}
	}
	e.domainPageHits[domain][keyword]++
}

func (e *Extractor) Count() int     { return e.sessionLen }
func (e *Extractor) Unique() int    { return len(e.scores) }
func (e *Extractor) Stopwords() int { return len(e.filter.stopwords) }

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
	seenTok := map[string]struct{}{}
	for _, t := range tokens {
		for _, part := range expandCompoundToken(t) {
			if !e.filter.AcceptToken(part) {
				continue
			}
			if _, ok := seenTok[part]; ok {
				continue
			}
			seenTok[part] = struct{}{}
			valid = append(valid, part)
			mergeCandidate(out, part, blk.weight, "token")
		}
	}

	if isHeading(blk.tag) && e.filter.AcceptPhrase(text) {
		mergeCandidate(out, text, blk.weight*2, "heading")
	}

	for i := 0; i < len(valid); i++ {
		if i+1 < len(valid) {
			bigram := valid[i] + " " + valid[i+1]
			mergeCandidate(out, bigram, pairWeight(blk.weight, 2), "bigram")
		}
		if i+2 < len(valid) && blk.weight >= 3 {
			trigram := valid[i] + " " + valid[i+1] + " " + valid[i+2]
			mergeCandidate(out, trigram, pairWeight(blk.weight, 3), "trigram")
		}
	}

	if blk.weight >= 3 && len(valid) >= 2 {
		limit := min(len(valid), 8)
		for i := 0; i < limit; i++ {
			for j := i + 1; j < limit && j < i+3; j++ {
				pair := valid[i] + " " + valid[j]
				mergeCandidate(out, pair, pairWeight(blk.weight, 2), "pair")
			}
		}
	}
}

func collectMeta(doc *html.Node) []block {
	var blocks []block
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, prop, content string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "name":
					name = strings.ToLower(a.Val)
				case "property":
					prop = strings.ToLower(a.Val)
				case "content":
					content = a.Val
				}
			}
			key := name
			if prop != "" {
				key = prop
			}
			switch key {
			case "description", "keywords", "og:title", "og:description", "twitter:title", "twitter:description":
				if strings.TrimSpace(content) != "" {
					blocks = append(blocks, block{tag: "meta", weight: 4, text: content})
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

func collectBlocks(doc *html.Node) []block {
	var blocks []block
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if w, ok := tagWeights[n.Data]; ok {
				text := deepText(n)
				if strings.TrimSpace(text) != "" {
					source := n.Data
					if source == "meta" {
						source = "meta"
					}
					blocks = append(blocks, block{tag: source, weight: w, text: text})
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

func (e *Extractor) addPathScore(domain, phrase string, weight int) {
	if e.domainPathScores[domain] == nil {
		e.domainPathScores[domain] = map[string]int{}
	}
	e.domainPathScores[domain][phrase] += weight
}

// TopPathsForDomain returns frequent URL path tokens from crawled pages.
func (e *Extractor) TopPathsForDomain(domain string, limit int) []string {
	scores := e.domainPathScores[domain]
	if len(scores) == 0 {
		return nil
	}
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
	out := make([]string, len(arr))
	for i, item := range arr {
		out[i] = item.k
	}
	return out
}

func (e *Extractor) addDomainScore(domain, phrase string, weight int) {
	if e.domainScores[domain] == nil {
		e.domainScores[domain] = map[string]int{}
	}
	e.domainScores[domain][phrase] += weight
}

// Top returns the highest-scoring keywords accumulated in the session.
func (e *Extractor) Top(limit int) []Result {
	return e.topIntelligent(e.scores, nil, limit)
}

// TopForDomain returns the best recon keywords for a domain using intelligent ranking.
func (e *Extractor) TopForDomain(domain string, limit int) []Result {
	if e.domainScores[domain] == nil {
		return nil
	}
	results := e.topIntelligent(e.domainScores[domain], e.domainPageHits[domain], limit)
	for i := range results {
		results[i].Domain = domain
	}
	return results
}

func (e *Extractor) topIntelligent(scores map[string]int, pageHits map[string]int, limit int) []Result {
	type ranked struct {
		keyword string
		score   float64
		weight  int
	}
	arr := make([]ranked, 0, len(scores))
	for k, v := range scores {
		hits := 0
		if pageHits != nil {
			hits = pageHits[k]
		}
		arr = append(arr, ranked{
			keyword: k,
			score:   intelligentRank(v, hits, k),
			weight:  v,
		})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].score == arr[j].score {
			return arr[i].keyword < arr[j].keyword
		}
		return arr[i].score > arr[j].score
	})

	var picked []ranked
	for _, item := range arr {
		skip := false
		for _, p := range picked {
			if isSubsumedBy(item.keyword, p.keyword) && item.score <= p.score*1.1 {
				skip = true
				break
			}
			if isSubsumedBy(p.keyword, item.keyword) && p.score <= item.score*1.1 {
				for j := range picked {
					if picked[j].keyword == p.keyword {
						picked = append(picked[:j], picked[j+1:]...)
						break
					}
				}
			}
		}
		if skip {
			continue
		}
		picked = append(picked, item)
		if limit > 0 && len(picked) >= limit {
			break
		}
	}

	out := make([]Result, len(picked))
	for i, item := range picked {
		out[i] = Result{Keyword: item.keyword, Weight: item.weight}
	}
	return out
}
