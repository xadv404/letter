package dorks

import (
	"strings"
	"testing"
)

func TestAssembleFromMaterials(t *testing.T) {
	fp := NewFingerprint()
	fp.AddParameter("id")
	fp.AddParameter("cat")
	fp.AddPath("view")
	fp.AddPath("catalog")
	fp.Finalize()
	m := PrepareMaterials(*fp, []string{"catalog", "wholesale"}, []string{"invoice portal"})
	m.ParamScores = map[string]int{"id": 95, "cat": 80}

	out := Assemble(m)
	if len(out) < 10 {
		t.Fatalf("expected 10+ assembled dorks, got %d", len(out))
	}
	if len(out) > MaxAssembledDorks {
		t.Fatalf("too many dorks %d", len(out))
	}
	for _, a := range out {
		if strings.Contains(a.Dork, "{") {
			t.Fatalf("unfilled placeholder: %s", a.Dork)
		}
		if strings.Contains(a.Dork, "site:") {
			t.Fatalf("must not contain site: %s", a.Dork)
		}
	}
}

func TestAssembleNoDuplicate(t *testing.T) {
	m := Materials{
		Types:    AllDorkTypes(),
		Keywords: []string{"admin", "catalog"},
		Params:   []string{"id", "cat"},
		ParamScores: map[string]int{"id": 90, "cat": 75},
	}
	seen := map[string]bool{}
	for _, a := range Assemble(m) {
		if seen[a.Dork] {
			t.Fatalf("duplicate dork: %s", a.Dork)
		}
		seen[a.Dork] = true
	}
}

func TestAssembleKeywordClone(t *testing.T) {
	m := Materials{
		Types:         AllDorkTypes(),
		Keywords:      []string{"wholesale"},
		Params:        []string{"id"},
		ParamScores:   map[string]int{"id": 88},
		KeywordScores: map[string]int{"wholesale": 30},
	}
	found := false
	for _, a := range Assemble(m) {
		if a.Dork == "inurl:id= intext:wholesale" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected clone dork inurl:id= intext:wholesale")
	}
}

func TestAssembleFiltersNoiseParams(t *testing.T) {
	m := Materials{
		Types: AllDorkTypes(),
		Params: []string{"id", "_gl", "contextualppvid", "signup", "player_version", "cat"},
		ParamScores: map[string]int{
			"id": 95, "_gl": 70, "contextualppvid": 72, "signup": 68, "player_version": 66, "cat": 80,
		},
	}
	for _, a := range Assemble(m) {
		if isNoiseParam(a.Param) {
			t.Fatalf("noise param leaked into dorks: %s (%s)", a.Param, a.Dork)
		}
	}
}

func TestMaxTwoVolumeDorksPerParam(t *testing.T) {
	m := Materials{
		Types:       AllDorkTypes(),
		Params:      []string{"id"},
		ParamScores: map[string]int{"id": 95},
		Paths:       []string{"product", "index", "news", "item"},
	}
	count := 0
	for _, a := range Assemble(m) {
		if a.Family == "param_volume" && a.Param == "id" {
			count++
		}
	}
	if count > MaxVolumeDorksPerParam {
		t.Fatalf("expected max %d volume dorks for id, got %d", MaxVolumeDorksPerParam, count)
	}
}

func TestAssembleCapped(t *testing.T) {
	m := Materials{
		Types:    AllDorkTypes(),
		Keywords: make([]string, 100),
		Params:   make([]string, 100),
	}
	for i := range m.Keywords {
		m.Keywords[i] = "kw" + string(rune('a'+i%26))
	}
	for i := range m.Params {
		m.Params[i] = "param" + string(rune('a'+i%26))
	}
	if len(Assemble(m)) > MaxAssembledDorks {
		t.Fatalf("should cap at %d", MaxAssembledDorks)
	}
}
