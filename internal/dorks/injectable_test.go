package dorks

import "testing"

func TestExpandInjectableParams(t *testing.T) {
	out := ExpandInjectableParams([]string{"product_id", "search_term"})
	seen := map[string]bool{}
	for _, p := range out {
		seen[p] = true
	}
	for _, want := range []string{"product_id", "product", "id", "pid", "search_term", "q", "cat", "catid"} {
		if !seen[want] {
			t.Fatalf("missing expanded param %q in %#v", want, out)
		}
	}
}

func TestIsInjectableParam(t *testing.T) {
	if !IsInjectableParam("id", 95) {
		t.Fatal("id should be injectable")
	}
	if !IsInjectableParam("cat", 55) {
		t.Fatal("cat should be injectable at score 55")
	}
	if IsInjectableParam("utm_source", 10) {
		t.Fatal("utm_source should not be injectable")
	}
}
