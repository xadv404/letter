package dorks

import (
	"strings"
	"testing"
)

func TestNoSeedHostInDorks(t *testing.T) {
	g := New()
	seed := "victim-secret-shop.com"
	out := g.Generate(Options{
		Host:       seed,
		TLD:        ".com",
		Keywords:   []string{"admin", "catalog"},
		Parameters: []string{"id", "cat"},
	})
	if len(out) == 0 {
		t.Fatal("expected dorks")
	}
	for _, d := range out {
		if strings.Contains(d, seed) {
			t.Fatalf("dork must not reference crawled host: %s", d)
		}
		if strings.Contains(d, "site:") {
			t.Fatalf("dork must not contain site: operator: %s", d)
		}
	}
}

func TestGenerateDorksMatrix(t *testing.T) {
	g := New()
	out := g.Generate(Options{
		Host:       "shop.com",
		TLD:        ".com",
		Keywords:   []string{"admin", "user"},
		Parameters: []string{"id", "search_term"},
		PreviewLimit: 10,
	})
	if len(out) == 0 {
		t.Fatal("expected dorks")
	}
	found := false
	for _, d := range out {
		if strings.Contains(d, "inurl:id= intext:admin") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected matrix dork inurl:id= intext:admin, got %#v", out[:min(3, len(out))])
	}
}

func TestGenerateUnique(t *testing.T) {
	g := New()
	opts := Options{Host: "example.com", TLD: ".com", Keywords: []string{"admin"}, Parameters: []string{"id"}}
	first := g.Generate(opts)
	second := g.Generate(opts)
	if len(second) != 0 {
		t.Fatalf("expected no duplicate dorks, got %d", len(second))
	}
	if len(first) == 0 {
		t.Fatal("expected initial dorks")
	}
}

func TestTemplateCountAtLeast50(t *testing.T) {
	if n := TemplateCount(); n < 50 {
		t.Fatalf("expected at least 50 templates, got %d", n)
	}
}

func TestSQLiErrorDorkGenerated(t *testing.T) {
	g := New()
	out := g.Generate(Options{
		Keywords:   []string{"catalog"},
		Parameters: []string{"id"},
	})
	found := false
	for _, d := range out {
		if strings.Contains(d, `intext:"You have an error in your SQL syntax" inurl:id=`) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected SQL error leak dork")
	}
}

func TestSQLiEndpointDorkGenerated(t *testing.T) {
	g := New()
	out := g.Generate(Options{
		Keywords:   []string{"product"},
		Parameters: []string{"cat"},
	})
	found := false
	for _, d := range out {
		if strings.Contains(d, "inurl:product.php inurl:cat=") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected classic endpoint dork inurl:product.php inurl:cat=")
	}
}

func TestSiteScope(t *testing.T) {
	host, tld := SiteScope("https://shop.example.com/path")
	if host != "shop.example.com" || tld != ".com" {
		t.Fatalf("unexpected scope: host=%s tld=%s", host, tld)
	}
}

func TestPreviewFormat(t *testing.T) {
	out := Preview(Options{
		Host: "shop.com", TLD: ".com",
		Keywords: []string{"admin"}, Parameters: []string{"id"},
	}, 3)
	if !strings.Contains(out, "inurl:id=") {
		t.Fatalf("unexpected preview: %s", out)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
