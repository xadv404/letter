package dorks

import (
	"net/url"
	"sort"
	"strings"
)

const (
	maxKeywords   = 80
	maxPhrases    = 40
	maxParameters = 120
	maxPaths      = 80
)

// Fingerprint holds crawl-derived signals used to find unknown clone sites.
type Fingerprint struct {
	Keywords   []string
	Phrases    []string
	Parameters []string
	Paths      []string

	kwSeen   map[string]struct{}
	phSeen   map[string]struct{}
	pmSeen   map[string]struct{}
	pathSeen map[string]struct{}
}

func NewFingerprint() *Fingerprint {
	return &Fingerprint{
		kwSeen:   map[string]struct{}{},
		phSeen:   map[string]struct{}{},
		pmSeen:   map[string]struct{}{},
		pathSeen: map[string]struct{}{},
	}
}

func (f *Fingerprint) AddTerm(term string) {
	term = normalizeTerm(term)
	if term == "" {
		return
	}
	if strings.Contains(term, " ") {
		if _, ok := f.phSeen[term]; ok {
			return
		}
		f.phSeen[term] = struct{}{}
		f.Phrases = append(f.Phrases, term)
		return
	}
	if _, ok := f.kwSeen[term]; ok {
		return
	}
	f.kwSeen[term] = struct{}{}
	f.Keywords = append(f.Keywords, term)
}

func (f *Fingerprint) AddParameter(name string) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return
	}
	if _, ok := f.pmSeen[name]; ok {
		return
	}
	f.pmSeen[name] = struct{}{}
	f.Parameters = append(f.Parameters, name)
}

func (f *Fingerprint) AddPath(path string) {
	path = strings.ToLower(strings.TrimSpace(path))
	path = strings.Trim(path, "/")
	if path == "" || len(path) < 2 {
		return
	}
	if _, ok := f.pathSeen[path]; ok {
		return
	}
	f.pathSeen[path] = struct{}{}
	f.Paths = append(f.Paths, path)
}

// AddURLPaths extracts directory and file tokens from a crawled URL.
func (f *Fingerprint) AddURLPaths(rawURL string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return
	}
	parts := strings.Split(path, "/")
	for _, seg := range parts {
		seg = strings.ToLower(strings.TrimSpace(seg))
		if seg == "" {
			continue
		}
		f.AddPath(seg)
		if i := strings.LastIndex(seg, "."); i > 0 {
			f.AddPath(seg[:i])
		}
	}
}

func (f *Fingerprint) Finalize() {
	f.Phrases = capStrings(f.Phrases, maxPhrases)
	f.Keywords = capStrings(f.Keywords, maxKeywords)
	f.Parameters = capStrings(f.Parameters, maxParameters)
	f.Paths = capStrings(f.Paths, maxPaths)
	sort.Strings(f.Keywords)
	sort.Strings(f.Phrases)
	sort.Strings(f.Parameters)
	sort.Strings(f.Paths)
}

func (f *Fingerprint) Viable() bool {
	return len(f.Parameters) > 0
}

func normalizeTerm(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

func capStrings(in []string, max int) []string {
	if max <= 0 || len(in) <= max {
		return in
	}
	return in[:max]
}
