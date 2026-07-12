package dorks

import "testing"

func TestGenerateURLVolumeCount(t *testing.T) {
	set := GenerateURLVolume([]string{"order_id", "news_id"})
	if n := len(set.All); n < 40 || n > MaxURLVolumeDorks {
		t.Fatalf("expected 40-%d dorks, got %d", MaxURLVolumeDorks, n)
	}
}

func TestURLVolumeDorksAreBroad(t *testing.T) {
	set := GenerateURLVolume(nil)
	for _, d := range set.All {
		if stringsContains(d, "intext:") {
			t.Fatalf("volume dork must not use intext: %s", d)
		}
		if countInurl(d) > 1 {
			t.Fatalf("volume dork must use single inurl: %s", d)
		}
	}
}

func TestURLVolumeIncludesCore(t *testing.T) {
	set := GenerateURLVolume(nil)
	seen := map[string]bool{}
	for _, d := range set.All {
		seen[d] = true
	}
	for _, want := range []string{"inurl:id=", "inurl:.php?id=", "inurl:view.php?id="} {
		if !seen[want] {
			t.Fatalf("missing core volume dork %q", want)
		}
	}
}

func TestCrawledParamsInjected(t *testing.T) {
	set := GenerateURLVolume([]string{"invoice_id"})
	found := false
	for _, d := range set.All {
		if d == "inurl:invoice_id=" || d == "inurl:.php?invoice_id=" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected crawled invoice_id in volume dorks")
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || findSub(s, sub))
}

func findSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func countInurl(d string) int {
	n := 0
	for i := 0; i < len(d); i++ {
		if i+5 <= len(d) && d[i:i+5] == "inurl" {
			n++
		}
	}
	return n
}
