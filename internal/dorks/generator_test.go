package dorks

import "testing"

func TestGenerateDorks(t *testing.T) {
	g := New()
	out := g.Generate("example.com", []string{"invoice"}, []string{"id"}, 3)
	if len(out) == 0 {
		t.Fatal("expected dorks")
	}
	if out[0] == "" {
		t.Fatal("empty dork")
	}
}

func TestGenerateUnique(t *testing.T) {
	g := New()
	first := g.Generate("example.com", []string{"admin"}, []string{"id"}, 0)
	second := g.Generate("example.com", []string{"admin"}, []string{"id"}, 0)
	if len(second) != 0 {
		t.Fatalf("expected no duplicate dorks, got %d", len(second))
	}
	if len(first) == 0 {
		t.Fatal("expected initial dorks")
	}
}
