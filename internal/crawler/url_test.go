package crawler

import "testing"

func TestCanonicalURL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"https://Example.com/page/", "https://example.com/page"},
		{"https://example.com/page#section", "https://example.com/page"},
		{"https://example.com/", "https://example.com/"},
	}
	for _, tc := range tests {
		if got := canonicalURL(tc.in); got != tc.want {
			t.Fatalf("canonicalURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
