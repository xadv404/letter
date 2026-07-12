package keywords

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func parseHTML(raw string) *html.Node {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return doc
}

func TestStopwordCountOver1000(t *testing.T) {
	if StopwordCount() < 1000 {
		t.Fatalf("expected 1000+ stopwords, got %d", StopwordCount())
	}
}

func TestDeepTextExtractionNestedHeading(t *testing.T) {
	doc := parseHTML(`<html><body><h1><span>SQL</span> <em>Injection</em> Guide</h1></body></html>`)
	e := New(1000)
	results := e.Extract("example.com", doc)

	found := map[string]bool{}
	for _, r := range results {
		found[r.Keyword] = true
	}
	if !found["sql injection guide"] {
		t.Fatalf("expected heading phrase, got %#v", results)
	}
	if !found["sql"] || !found["injection"] {
		t.Fatalf("expected token keywords, got %#v", results)
	}
}

func TestSkipsScriptAndStyle(t *testing.T) {
	doc := parseHTML(`<html><body><p>real keyword</p><script>var admin_panel=true</script><style>.hidden{}</style></body></html>`)
	e := New(1000)
	results := e.Extract("example.com", doc)
	for _, r := range results {
		if strings.Contains(r.Keyword, "admin_panel") || strings.Contains(r.Keyword, "hidden") {
			t.Fatalf("unexpected technical noise: %s", r.Keyword)
		}
	}
}

func TestAdjacentBigram(t *testing.T) {
	doc := parseHTML(`<html><body><p>database backup restore procedure</p></body></html>`)
	e := New(1000)
	results := e.Extract("example.com", doc)
	found := false
	for _, r := range results {
		if r.Keyword == "database backup" && r.Source == "bigram" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected adjacent bigram, got %#v", results)
	}
}

func TestStopwordRejected(t *testing.T) {
	f := NewFilter()
	if f.AcceptToken("the") || f.AcceptToken("and") {
		t.Fatal("stopwords should be rejected")
	}
}

func TestTechnicalRejected(t *testing.T) {
	f := NewFilter()
	if f.AcceptToken("javascript") || f.AcceptToken("bootstrap") {
		t.Fatal("technical terms should be rejected")
	}
}

func TestCumulativeScoringAcrossPages(t *testing.T) {
	e := New(1000)
	doc1 := parseHTML(`<html><body><h1>Invoice Portal</h1></body></html>`)
	doc2 := parseHTML(`<html><body><h2>Invoice Management</h2></body></html>`)

	r1 := e.Extract("example.com", doc1)
	r2 := e.Extract("example.com", doc2)

	var invoiceWeight int
	for _, r := range r2 {
		if r.Keyword == "invoice" {
			invoiceWeight = r.Weight
		}
	}
	if len(r1) == 0 || invoiceWeight <= 5 {
		t.Fatalf("expected cumulative score for 'invoice', r1=%#v r2=%#v", r1, r2)
	}
}

func TestSessionCap(t *testing.T) {
	e := New(3)
	doc := parseHTML(`<html><body>
		<h1>alpha beta gamma</h1>
		<p>delta epsilon zeta eta theta</p>
	</body></html>`)
	_ = e.Extract("example.com", doc)
	if e.Count() > 3 {
		t.Fatalf("session cap exceeded: %d", e.Count())
	}
}

func TestPhraseLengthConstraints(t *testing.T) {
	f := NewFilter()
	if f.AcceptToken("a") {
		t.Fatal("too short")
	}
	long := strings.Repeat("x", 101)
	if f.AcceptToken(long) {
		t.Fatal("too long")
	}
}

func TestHeadingHigherWeightThanParagraph(t *testing.T) {
	e := New(1000)
	doc := parseHTML(`<html><body><h1>vulnerability</h1><p>vulnerability</p></body></html>`)
	results := e.Extract("example.com", doc)
	var weight int
	for _, r := range results {
		if r.Keyword == "vulnerability" {
			weight = r.Weight
		}
	}
	if weight < 6 {
		t.Fatalf("expected cumulative weight >= 6 (5+1), got %d", weight)
	}
}

func TestTopForDomain(t *testing.T) {
	e := New(1000)
	e.Extract("a.com", parseHTML(`<html><body><h1>invoice portal</h1></body></html>`))
	e.Extract("b.com", parseHTML(`<html><body><h1>payment gateway</h1></body></html>`))

	topA := e.TopForDomain("a.com", 5)
	if len(topA) == 0 || topA[0].Keyword == "payment" {
		t.Fatalf("expected a.com keywords, got %#v", topA)
	}
}
