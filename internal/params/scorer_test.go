package params

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScoreHighRiskParam(t *testing.T) {
	s := New(1000, t.TempDir())
	score, tier, _, _ := s.evaluate("search_term")
	if tier != TierHigh {
		t.Fatalf("expected HIGH tier, got %s", tier)
	}
	if score < 85 {
		t.Fatalf("expected score >= 85, got %d", score)
	}
}

func TestHighTierExactParams(t *testing.T) {
	s := New(1000, t.TempDir())
	cases := []string{"id", "filter_by", "sort", "search_term"}
	for _, name := range cases {
		score, tier, _, _ := s.evaluate(name)
		if tier != TierHigh || score < 85 {
			t.Fatalf("%s: expected HIGH >= 85, got tier=%s score=%d", name, tier, score)
		}
	}
}

func TestExcludeTrackingParam(t *testing.T) {
	s := New(1000, t.TempDir())
	score, tier, _, reason := s.evaluate("utm_source")
	if tier != TierExclude {
		t.Fatalf("expected EXCLUDE tier, got %s", tier)
	}
	if score >= 50 {
		t.Fatalf("expected score < 50, got %d", score)
	}
	if reason == "" {
		t.Fatal("expected exclusion reason")
	}
}

func TestExcludeSessionToken(t *testing.T) {
	s := New(1000, t.TempDir())
	_, tier, _, _ := s.evaluate("phpsessid")
	if tier != TierExclude {
		t.Fatalf("expected EXCLUDE for session token, got %s", tier)
	}
}

func TestSecListsMatch(t *testing.T) {
	s := New(1000, t.TempDir())
	if s.SecListsSize() < 1000 {
		t.Fatalf("expected 1000+ SecLists params, got %d", s.SecListsSize())
	}
	score, tier, matched, _ := s.evaluate("username")
	if matched != "seclists" {
		t.Fatalf("expected seclists match, got %s", matched)
	}
	if tier != TierMedium || score < 65 || score > 84 {
		t.Fatalf("expected MEDIUM 65-84, got tier=%s score=%d", tier, score)
	}
}

func TestLowTierExcludedByDefault(t *testing.T) {
	s := New(1000, t.TempDir())
	score, tier, _, _ := s.evaluate("xy")
	if tier != TierLow {
		t.Fatalf("expected LOW tier, got %s (score=%d)", tier, score)
	}
	if score < 50 || score > 64 {
		t.Fatalf("expected score 50-64, got %d", score)
	}
	results := s.ScoreURL("example.com", "https://example.com/?xy=1", 65)
	if len(results) != 0 {
		t.Fatalf("LOW tier should be excluded at minScore 65, got %d results", len(results))
	}
}

func TestScoreURLFiltersByMinScore(t *testing.T) {
	s := New(1000, t.TempDir())
	results := s.ScoreURL("example.com", "https://example.com/page?utm_source=1&id=5", 65)
	if len(results) != 1 {
		t.Fatalf("expected 1 accepted param, got %d", len(results))
	}
	if results[0].Name != "id" {
		t.Fatalf("expected id param, got %s", results[0].Name)
	}
	if results[0].Score < 85 {
		t.Fatalf("expected HIGH score for id, got %d", results[0].Score)
	}
}

func TestDeduplicationPerDomain(t *testing.T) {
	s := New(1000, t.TempDir())
	r1 := s.ScoreURL("example.com", "https://example.com/a?id=1", 65)
	r2 := s.ScoreURL("example.com", "https://example.com/b?id=2", 65)
	if len(r1) != 1 || len(r2) != 0 {
		t.Fatalf("expected dedup per domain+param, r1=%d r2=%d", len(r1), len(r2))
	}
	if s.Count() != 1 {
		t.Fatalf("expected count 1, got %d", s.Count())
	}
}

func TestTopForDomain(t *testing.T) {
	s := New(1000, t.TempDir())
	s.ScoreURL("example.com", "https://example.com/?id=1&search_term=test", 65)
	top := s.TopForDomain("example.com", 10)
	if len(top) < 2 {
		t.Fatalf("expected 2+ params, got %d", len(top))
	}
	if top[0].Score < top[len(top)-1].Score {
		t.Fatal("expected descending score order")
	}
}

func TestWordlistCache(t *testing.T) {
	dir := t.TempDir()
	s1 := New(100, dir)
	size := s1.SecListsSize()

	cachePath := filepath.Join(dir, cacheFileName)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("expected cache file to be written")
	}

	s2 := New(100, dir)
	if s2.SecListsSize() != size {
		t.Fatalf("cache reload size mismatch: %d vs %d", s2.SecListsSize(), size)
	}
}

func TestMediumPatternScoring(t *testing.T) {
	s := New(1000, t.TempDir())
	score, tier, matched, reason := s.evaluate("customer_id")
	if tier != TierHigh {
		// customer_id ends with _id suffix -> HIGH
		t.Fatalf("customer_id: expected HIGH, got %s score=%d matched=%s", tier, score, matched)
	}
	if score < 85 {
		t.Fatalf("expected suffix high score, got %d", score)
	}

	score, tier, matched, reason = s.evaluate("report_type")
	if tier != TierMedium || score < 65 {
		t.Fatalf("report_type: expected MEDIUM, got %s score=%d", tier, score)
	}
	if matched != "pattern" && matched != "seclists" {
		t.Fatalf("unexpected match source: %s reason=%s", matched, reason)
	}
}

func TestTierBoundaries(t *testing.T) {
	cases := []struct {
		score int
		tier  Tier
	}{
		{95, TierHigh},
		{85, TierHigh},
		{84, TierMedium},
		{65, TierMedium},
		{64, TierLow},
		{50, TierLow},
		{49, TierExclude},
	}
	for _, c := range cases {
		if got := scoreToTier(c.score); got != c.tier {
			t.Fatalf("score %d: expected %s, got %s", c.score, c.tier, got)
		}
	}
}
