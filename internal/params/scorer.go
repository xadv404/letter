package params

import (
	"net/url"
	"strings"
	"sync"
)

type Tier string

const (
	TierHigh    Tier = "HIGH"    // score ≥ 85
	TierMedium  Tier = "MEDIUM"  // score 65–84
	TierLow     Tier = "LOW"     // score 50–64
	TierExclude Tier = "EXCLUDE" // score < 50
)

const MinDorkScore = 65 // HIGH + MEDIUM only for dork generation

type Result struct {
	Domain  string
	URL     string
	Name    string
	Score   int
	Tier    Tier
	Matched string
}

type FlaggedURL struct {
	Domain     string
	URL        string
	HighParams []string
	MaxScore   int
}

type FilterDecision struct {
	Param    string
	Score    int
	Tier     Tier
	Accepted bool
	Reason   string
}

type Scorer struct {
	mu          sync.Mutex
	seclists    map[string]struct{}
	maxCount    int
	count       int
	seen        map[string]struct{}
	decisions   []FilterDecision
	stats       struct {
		Accepted int
		Rejected int
	}
	domainHits  map[string]map[string]int
	flaggedURLs map[string]map[string]*FlaggedURL
}

func New(maxCount int, cacheDir string) *Scorer {
	return &Scorer{
		seclists:    LoadWordlist(cacheDir),
		maxCount:    maxCount,
		seen:        map[string]struct{}{},
		domainHits:  map[string]map[string]int{},
		flaggedURLs: map[string]map[string]*FlaggedURL{},
	}
}

func (s *Scorer) SecListsSize() int { return len(s.seclists) }

func (s *Scorer) ScoreURL(domain, rawURL string, minScore int) []Result {
	u, err := url.Parse(rawURL)
	if err != nil || u.RawQuery == "" {
		return nil
	}
	vals, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil
	}

	var out []Result
	var highParams []string
	maxScore := 0

	for name := range vals {
		if s.count >= s.maxCount {
			break
		}
		key := domain + "\x00" + name
		s.mu.Lock()
		already := false
		if _, dup := s.seen[key]; dup {
			already = true
			if sc, ok := s.domainHits[domain][name]; ok && sc >= 85 {
				highParams = appendUnique(highParams, name)
				if sc > maxScore {
					maxScore = sc
				}
			}
		}
		s.mu.Unlock()
		if already {
			continue
		}

		score, tier, matched, reason := s.evaluate(name)
		accepted := score >= minScore
		s.recordDecision(name, score, tier, accepted, reason)

		if !accepted {
			continue
		}

		s.mu.Lock()
		s.seen[key] = struct{}{}
		s.count++
		if s.domainHits[domain] == nil {
			s.domainHits[domain] = map[string]int{}
		}
		s.domainHits[domain][name] = score
		s.mu.Unlock()

		if score > maxScore {
			maxScore = score
		}
		if tier == TierHigh {
			highParams = appendUnique(highParams, name)
		}

		out = append(out, Result{
			Domain:  domain,
			URL:     rawURL,
			Name:    name,
			Score:   score,
			Tier:    tier,
			Matched: matched,
		})
	}

	if len(highParams) > 0 {
		s.flagURL(domain, rawURL, highParams, maxScore)
	}
	return out
}

func (s *Scorer) flagURL(domain, rawURL string, highParams []string, maxScore int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.flaggedURLs[domain] == nil {
		s.flaggedURLs[domain] = map[string]*FlaggedURL{}
	}
	existing := s.flaggedURLs[domain][rawURL]
	if existing == nil {
		existing = &FlaggedURL{Domain: domain, URL: rawURL, HighParams: highParams, MaxScore: maxScore}
		s.flaggedURLs[domain][rawURL] = existing
		return
	}
	for _, p := range highParams {
		existing.HighParams = appendUnique(existing.HighParams, p)
	}
	if maxScore > existing.MaxScore {
		existing.MaxScore = maxScore
	}
}

func (s *Scorer) evaluate(name string) (score int, tier Tier, matched, reason string) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return 0, TierExclude, "empty", "empty parameter name"
	}

	if exScore, exReason, ok := matchExclude(lower); ok {
		return exScore, TierExclude, "exclude_rule", exReason
	}

	// LOW before SecLists — page/sort/category stay weak even if in wordlist.
	if sc, ok := scoreLowExact(lower); ok {
		return sc, TierLow, "weak", "weak pagination/sort/CMS-like parameter"
	}

	if sc, ok := scoreHighExact(lower); ok {
		return sc, TierHigh, "high_exact", "classically injectable SQLi parameter"
	}
	if sc, ok := scoreHighSuffix(lower); ok {
		return sc, TierHigh, "high_suffix", "high-risk SQLi parameter (id/num suffix)"
	}

	if _, ok := s.seclists[lower]; ok {
		return 78, TierMedium, "seclists", "verified against SecLists Burp parameter wordlist"
	}

	if sc, patReason, ok := scoreMediumPattern(lower); ok {
		return sc, scoreToTier(sc), "pattern", patReason
	}

	if isPlausibleParam(lower) {
		return 68, TierMedium, "unknown", "unknown parameter — plausible SQLi candidate"
	}

	return 45, TierExclude, "noise", "no SQLi relevance detected"
}

func scoreToTier(score int) Tier {
	switch {
	case score >= 85:
		return TierHigh
	case score >= 65:
		return TierMedium
	case score >= 50:
		return TierLow
	default:
		return TierExclude
	}
}

func isPlausibleParam(name string) bool {
	if len(name) < 3 || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func (s *Scorer) recordDecision(param string, score int, tier Tier, accepted bool, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if accepted {
		s.stats.Accepted++
	} else {
		s.stats.Rejected++
	}
	s.decisions = append(s.decisions, FilterDecision{
		Param: param, Score: score, Tier: tier, Accepted: accepted, Reason: reason,
	})
	if len(s.decisions) > 100 {
		s.decisions = s.decisions[len(s.decisions)-100:]
	}
}

func (s *Scorer) Decisions() []FilterDecision {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]FilterDecision, len(s.decisions))
	copy(out, s.decisions)
	return out
}

func (s *Scorer) Stats() (accepted, rejected int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats.Accepted, s.stats.Rejected
}

func (s *Scorer) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

func (s *Scorer) TopForDomain(domain string, limit int) []Result {
	return s.topForDomain(domain, limit, 0)
}

// TopForDorks returns HIGH + MEDIUM parameters (score ≥ minDorkScore).
func (s *Scorer) TopForDorks(domain string, limit int) []Result {
	return s.topForDomain(domain, limit, MinDorkScore)
}

func (s *Scorer) topForDomain(domain string, limit, minScore int) []Result {
	s.mu.Lock()
	hits := s.domainHits[domain]
	s.mu.Unlock()
	if len(hits) == 0 {
		return nil
	}

	type kv struct {
		name  string
		score int
	}
	arr := make([]kv, 0, len(hits))
	for name, score := range hits {
		if minScore > 0 && score < minScore {
			continue
		}
		arr = append(arr, kv{name, score})
	}
	for i := 0; i < len(arr); i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[j].score > arr[i].score {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
	if limit > 0 && len(arr) > limit {
		arr = arr[:limit]
	}
	out := make([]Result, len(arr))
	for i, item := range arr {
		out[i] = Result{
			Domain: domain,
			Name:   item.name,
			Score:  item.score,
			Tier:   scoreToTier(item.score),
		}
	}
	return out
}

func (s *Scorer) FlaggedURLs(domain string) []FlaggedURL {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.flaggedURLs[domain]
	if len(m) == 0 {
		return nil
	}
	out := make([]FlaggedURL, 0, len(m))
	for _, f := range m {
		out = append(out, *f)
	}
	return out
}

func (s *Scorer) AllFlaggedURLs() []FlaggedURL {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []FlaggedURL
	for _, m := range s.flaggedURLs {
		for _, f := range m {
			out = append(out, *f)
		}
	}
	return out
}
