package dorks

import "testing"

func TestGlobalLiteralBlastSize(t *testing.T) {
	if n := len(globalLiteralBlast()); n < 200 {
		t.Fatalf("expected 200+ global blast dorks, got %d", n)
	}
}

func TestLiteralEndpointSingleInurl(t *testing.T) {
	out := applyLiteralEndpoints("id")
	found := false
	for _, d := range out {
		if d == "inurl:view.php?id=" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected single-inurl literal endpoint inurl:view.php?id=")
	}
}
