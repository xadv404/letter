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

type Result struct {
	Domain  string
	URL     string
	Name    string
	Score   int
	Tier    Tier
	Matched string
}

type FilterDecision struct {
	Param    string
	Score    int
	Tier     Tier
	Accepted bool
	Reason   string
}

type Scorer struct {
	mu       sync.Mutex
	seclists map[string]struct{}
	maxCount int
	count    int
	seen     map[string]struct{}
	decisions []FilterDecision
	stats    struct {
		Accepted int
		Rejected int
	}
	domainHits map[string]map[string]int
}

// New creates a scorer. cacheDir enables the 7-day SecLists wordlist cache.
func New(maxCount int, cacheDir string) *Scorer {
	return &Scorer{
		seclists:   LoadWordlist(cacheDir),
		maxCount:   maxCount,
		seen:       map[string]struct{}{},
		domainHits: map[string]map[string]int{},
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
	for name := range vals {
		if s.count >= s.maxCount {
			break
		}
		key := domain + "\x00" + name
		s.mu.Lock()
		if _, dup := s.seen[key]; dup {
			s.mu.Unlock()
			continue
		}
		s.mu.Unlock()

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

		out = append(out, Result{
			Domain:  domain,
			URL:     rawURL,
			Name:    name,
			Score:   score,
			Tier:    tier,
			Matched: matched,
		})
	}
	return out
}

func (s *Scorer) evaluate(name string) (score int, tier Tier, matched, reason string) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return 0, TierExclude, "empty", "empty parameter name"
	}

	// Tier 1: exclusions (<50)
	if exScore, exReason, ok := matchExclude(lower); ok {
		return exScore, TierExclude, "exclude_rule", exReason
	}

	// Tier 3: HIGH (≥85) — exact names and suffixes
	if sc, ok := scoreHighExact(lower); ok {
		return sc, TierHigh, "high_exact", "high-risk SQLi parameter (exact match)"
	}
	if sc, ok := scoreHighSuffix(lower); ok {
		return sc, TierHigh, "high_suffix", "high-risk SQLi parameter (id/num suffix)"
	}

	// SecLists Burp wordlist verification → MEDIUM-HIGH
	if _, ok := s.seclists[lower]; ok {
		return 82, TierMedium, "seclists", "verified against SecLists Burp parameter wordlist"
	}

	// Pattern-based MEDIUM
	if sc, patReason, ok := scoreMediumPattern(lower); ok {
		return sc, scoreToTier(sc), "pattern", patReason
	}

	// Unknown but plausible alphanumeric → MEDIUM baseline
	if isPlausibleParam(lower) {
		return 68, TierMedium, "unknown", "unknown parameter — plausible SQLi candidate"
	}

	// LOW tier weak matches
	if sc, ok := scoreLowExact(lower); ok {
		return sc, TierLow, "weak", "weak/generic parameter name"
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

// TopForDomain returns highest-scored unique parameters for a domain.
func (s *Scorer) TopForDomain(domain string, limit int) []Result {
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
