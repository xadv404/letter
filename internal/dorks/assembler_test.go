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

	out := Assemble(m)
	if len(out) < 50 {
		t.Fatalf("expected 50+ assembled dorks, got %d", len(out))
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
		Types:    AllDorkTypes(),
		Keywords: []string{"wholesale"},
		Params:   []string{"id"},
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
