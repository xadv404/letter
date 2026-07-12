package dorks

import "testing"

func TestRateDorkEliteParamVolume(t *testing.T) {
	d := AssembledDork{
		Dork:   "inurl:id=",
		Family: "param_volume",
		Volume: "high",
		Param:  "id",
	}
	m := Materials{ParamScores: map[string]int{"id": 95}}
	score, tier := RateDork(d, m)
	if score < 85 {
		t.Fatalf("expected high score for id param_volume, got %d", score)
	}
	if tier != TierElite && tier != TierHigh {
		t.Fatalf("expected ELITE or HIGH tier, got %s (score %d)", tier, score)
	}
}

func TestRateDorkSqliError(t *testing.T) {
	d := AssembledDork{
		Dork:   `intext:"You have an error in your SQL syntax" inurl:id=`,
		Family: "sqli_error",
		Param:  "id",
	}
	score, tier := RateDork(d, Materials{ParamScores: map[string]int{"id": 90}})
	if score < 90 {
		t.Fatalf("sqli_error should score ELITE, got %d %s", score, tier)
	}
	if tier != TierElite {
		t.Fatalf("expected ELITE, got %s", tier)
	}
}

func TestRankAssembledSorted(t *testing.T) {
	m := Materials{
		Types:    AllDorkTypes(),
		Keywords: []string{"wholesale"},
		Params:   []string{"id", "tracking"},
		ParamScores: map[string]int{
			"id":       95,
			"tracking": 30,
		},
		KeywordScores: map[string]int{"wholesale": 40},
	}
	ranked := RankAssembled(m)
	if len(ranked) < 2 {
		t.Fatal("expected ranked dorks")
	}
	if ranked[0].Score < ranked[len(ranked)-1].Score {
		t.Fatalf("expected descending sort, first=%d last=%d", ranked[0].Score, ranked[len(ranked)-1].Score)
	}
	for _, d := range ranked {
		if d.Score == 0 || d.Tier == "" {
			t.Fatalf("missing rating on %q", d.Dork)
		}
	}
}

func TestKeywordCloneRated(t *testing.T) {
	m := Materials{
		Types:         AllDorkTypes(),
		Keywords:      []string{"wholesale"},
		Params:        []string{"id"},
		ParamScores:   map[string]int{"id": 88},
		KeywordScores: map[string]int{"wholesale": 35},
	}
	ranked := RankAssembled(m)
	for _, d := range ranked {
		if d.Dork == "inurl:id= intext:wholesale" {
			if d.Tier == TierLow {
				t.Fatalf("clone dork should not be LOW, got %d %s", d.Score, d.Tier)
			}
			return
		}
	}
	t.Fatal("expected clone dork")
}
