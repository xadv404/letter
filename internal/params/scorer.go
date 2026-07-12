package params

import (
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type Tier string

const (
	TierHigh    Tier = "HIGH"
	TierMedium  Tier = "MEDIUM"
	TierLow     Tier = "LOW"
	TierExclude Tier = "EXCLUDE"
)

type Result struct {
	Domain string
	URL    string
	Name   string
	Score  int
	Tier   Tier
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
	highPatterns []*regexp.Regexp
	excludePatterns []*regexp.Regexp
	maxCount    int
	count       int
	decisions   []FilterDecision
	stats       struct {
		Accepted int
		Rejected int
	}
}

func New(maxCount int) *Scorer {
	s := &Scorer{
		seclists: buildSecListsFallback(),
		maxCount: maxCount,
	}
	s.highPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(id|uid|user_id|product_id|item_id|order_id)$`),
		regexp.MustCompile(`(?i)^(search|search_term|q|query|keyword|keywords)$`),
		regexp.MustCompile(`(?i)^(filter|filter_by|sort|sort_by|order|orderby)$`),
		regexp.MustCompile(`(?i)^(cat|category|catid|page|pageid|article)$`),
	}
	s.excludePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^utm_`),
		regexp.MustCompile(`(?i)^(ga_|gclid|fbclid|mc_|pk_|_ga)$`),
		regexp.MustCompile(`(?i)^(csrf|nonce|token|session|sid|phpsessid|jsessionid)$`),
		regexp.MustCompile(`(?i)^(width|height|color|theme|lang|locale|currency)$`),
		regexp.MustCompile(`(?i)^(ref|referrer)$`),
	}
	return s
}

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
		score, tier := s.scoreParam(name)
		accepted := int(tierPriority(tier)) >= minScore
		reason := tierReason(tier, accepted)
		s.recordDecision(name, score, tier, accepted, reason)
		if !accepted {
			continue
		}
		s.mu.Lock()
		s.count++
		s.mu.Unlock()
		out = append(out, Result{
			Domain: domain,
			URL:    rawURL,
			Name:   name,
			Score:  score,
			Tier:   tier,
		})
	}
	return out
}

func (s *Scorer) scoreParam(name string) (int, Tier) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return 0, TierExclude
	}

	for _, re := range s.excludePatterns {
		if re.MatchString(lower) {
			return 30, TierExclude
		}
	}

	for _, re := range s.highPatterns {
		if re.MatchString(lower) {
			return 90, TierHigh
		}
	}

	if _, ok := s.seclists[lower]; ok {
		return 80, TierMedium
	}

	if strings.Contains(lower, "id") || strings.Contains(lower, "num") || strings.Contains(lower, "ref") {
		return 70, TierMedium
	}

	if len(lower) <= 3 {
		return 55, TierLow
	}

	return 68, TierMedium
}

func tierPriority(t Tier) int {
	switch t {
	case TierHigh:
		return 90
	case TierMedium:
		return 70
	case TierLow:
		return 55
	default:
		return 20
	}
}

func tierReason(t Tier, accepted bool) string {
	if accepted {
		return "score above threshold"
	}
	switch t {
	case TierExclude:
		return "tracking/CMS/analytics parameter"
	case TierLow:
		return "weak pattern match excluded by default"
	default:
		return "below minimum score"
	}
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

func buildSecListsFallback() map[string]struct{} {
	// Subset of common Burp/SecLists parameter names for offline use.
	names := []string{
		"id", "user", "username", "password", "email", "search", "query", "q", "page", "cat",
		"category", "product", "item", "order", "sort", "filter", "file", "path", "dir", "doc",
		"document", "view", "action", "cmd", "command", "exec", "module", "name", "type", "table",
		"column", "field", "select", "where", "report", "download", "export", "import", "debug",
		"test", "admin", "role", "group", "account", "invoice", "payment", "amount", "price",
		"year", "month", "day", "date", "from", "to", "limit", "offset", "start", "end",
	}
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}
