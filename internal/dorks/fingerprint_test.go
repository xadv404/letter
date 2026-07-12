package dorks

import (
	"strings"
	"testing"
)

func TestFingerprintViable(t *testing.T) {
	fp := NewFingerprint()
	fp.AddTerm("invoice portal")
	fp.AddParameter("id")
	fp.Finalize()
	if !fp.Viable() {
		t.Fatal("expected viable fingerprint")
	}
}

func TestGenerateFingerprintVolume(t *testing.T) {
	fp := NewFingerprint()
	fp.AddTerm("wholesale")
	fp.AddTerm("invoice payment")
	fp.AddParameter("id")
	fp.AddParameter("cat")
	fp.AddPath("catalog")
	fp.AddPath("view.php")
	fp.Finalize()

	g := New()
	out := g.GenerateFingerprint(*fp)
	if len(out) < 300 {
		t.Fatalf("expected 300+ injectable volume dorks, got %d", len(out))
	}
	paramOnly := 0
	for _, d := range out {
		if strings.HasPrefix(d, "inurl:id=") || strings.HasPrefix(d, "inurl:cat=") {
			paramOnly++
		}
	}
	if paramOnly < 20 {
		t.Fatalf("expected many param-only dorks, got %d", paramOnly)
	}
}

func TestGenerateFingerprintExpandedVolume(t *testing.T) {
	fp := NewFingerprint()
	for _, p := range ExpandInjectableParams([]string{"product_id", "cat", "search_term"}) {
		fp.AddParameter(p)
	}
	fp.AddPath("catalog")
	fp.AddPath("product")
	fp.Finalize()

	g := New()
	out := g.GenerateFingerprint(*fp)
	if len(out) < 3000 {
		t.Fatalf("expected 3000+ dorks with expanded injectable params, got %d", len(out))
	}
}

func TestFingerprintMergesSeeds(t *testing.T) {
	fp := NewFingerprint()
	fp.AddTerm("invoice")
	fp.AddParameter("id")
	fp.AddTerm("payment")
	fp.AddParameter("cat")
	fp.Finalize()
	if len(fp.Keywords) < 2 || len(fp.Parameters) < 2 {
		t.Fatalf("expected merged terms, got kw=%d pm=%d", len(fp.Keywords), len(fp.Parameters))
	}
}
