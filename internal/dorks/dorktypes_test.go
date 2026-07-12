package dorks

import (
	"strings"
	"testing"
)

func TestDorkTypeCount(t *testing.T) {
	n := DorkTypeCount()
	if n < 35 {
		t.Fatalf("expected 35+ dork types, got %d", n)
	}
}

func TestAllDorkTypesHavePlaceholders(t *testing.T) {
	for _, dt := range AllDorkTypes() {
		if dt.Pattern == "" || dt.ID == "" {
			t.Fatalf("invalid type: %#v", dt)
		}
		if len(dt.Slots) == 0 {
			t.Fatalf("type %s has no slots", dt.ID)
		}
	}
}

func TestPrepareMaterials(t *testing.T) {
	fp := NewFingerprint()
	fp.AddParameter("id")
	fp.AddParameter("cat")
	fp.AddPath("catalog")
	fp.Finalize()

	m := PrepareMaterials(*fp, []string{"wholesale", "invoice"}, []string{"invoice portal"})
	if len(m.Types) < 35 {
		t.Fatalf("expected types, got %d", len(m.Types))
	}
	if len(m.Keywords) < 2 {
		t.Fatal("expected keywords")
	}
	if len(m.Params) < 2 {
		t.Fatal("expected params")
	}
}

func TestDorkTypesAreNotAssembled(t *testing.T) {
	for _, dt := range AllDorkTypes() {
		if dt.Pattern == "inurl:id=" {
			t.Fatal("dork types must use placeholders, not hardcoded dorks")
		}
		if !strings.Contains(dt.Pattern, "{") {
			t.Fatalf("type %s must contain placeholder", dt.ID)
		}
	}
}
