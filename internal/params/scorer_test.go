package params

import "testing"

func TestScoreHighRiskParam(t *testing.T) {
	s := New(1000)
	score, tier := s.scoreParam("search_term")
	if tier != TierHigh {
		t.Fatalf("expected HIGH tier, got %s", tier)
	}
	if score < 85 {
		t.Fatalf("expected score >= 85, got %d", score)
	}
}

func TestExcludeTrackingParam(t *testing.T) {
	s := New(1000)
	_, tier := s.scoreParam("utm_source")
	if tier != TierExclude {
		t.Fatalf("expected EXCLUDE tier, got %s", tier)
	}
}

func TestScoreURLFiltersByMinScore(t *testing.T) {
	s := New(1000)
	results := s.ScoreURL("example.com", "https://example.com/page?utm_source=1&id=5", 65)
	if len(results) != 1 {
		t.Fatalf("expected 1 accepted param, got %d", len(results))
	}
	if results[0].Name != "id" {
		t.Fatalf("expected id param, got %s", results[0].Name)
	}
}
