package dorks

import "testing"

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
	if len(out) < 80 {
		t.Fatalf("expected 80+ dorks for theme clone hunting, got %d", len(out))
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
