package keywords

import "testing"

func TestExpandCompoundToken(t *testing.T) {
	parts := expandCompoundToken("productDetail")
	if len(parts) < 2 {
		t.Fatalf("expected camelCase split, got %#v", parts)
	}
}

func TestIntelligentRankPrefersPhrases(t *testing.T) {
	uni := intelligentRank(10, 2, "catalog")
	bi := intelligentRank(10, 2, "invoice payment")
	if bi <= uni {
		t.Fatalf("expected phrase to outrank token: uni=%v bi=%v", uni, bi)
	}
}

func TestReconMultiplier(t *testing.T) {
	if reconWeightMultiplier("invoice portal") <= reconWeightMultiplier("randomword") {
		t.Fatal("expected recon boost for invoice portal")
	}
}
