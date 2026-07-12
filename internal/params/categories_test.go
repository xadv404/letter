package params

import "testing"

func TestEightVulnerabilityCategories(t *testing.T) {
	cats := []Category{
		CatSQLi, CatXSS, CatRedirect, CatIDOR, CatLFI, CatSSRF, CatCmdInject, CatNoise,
	}
	if len(cats) != 8 {
		t.Fatalf("expected 8 categories, got %d", len(cats))
	}
}

func TestClassifySQLiParam(t *testing.T) {
	if classifyParam("search_term", "high_exact") != CatSQLi {
		t.Fatal("search_term should be sqli")
	}
}

func TestClassifyNoiseParam(t *testing.T) {
	if classifyParam("utm_source", "exclude_rule") != CatNoise {
		t.Fatal("utm_source should be noise")
	}
}

func TestWeightedScoreCaps(t *testing.T) {
	if weightedScore(95, CatSQLi) != 95 {
		t.Fatal("sqli weight should preserve high scores")
	}
	if weightedScore(50, CatNoise) >= 50 {
		t.Fatal("noise weight should reduce score")
	}
}
