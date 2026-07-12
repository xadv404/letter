package crawler

import "testing"

func TestNormalizeHost(t *testing.T) {
	if got := NormalizeHost("WWW.Example.COM"); got != "example.com" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeHost("www.shop.test"); got != "shop.test" {
		t.Fatalf("got %q", got)
	}
}
