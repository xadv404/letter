package dorks

import "testing"

func TestIsNoiseParam(t *testing.T) {
	noisy := []string{"_gl", "contextualppvid", "signup", "player_version", "samlrequest", "co", "lng"}
	for _, p := range noisy {
		if !isNoiseParam(p) {
			t.Fatalf("expected %q to be noise", p)
		}
	}
	if isNoiseParam("id") || isNoiseParam("product_id") {
		t.Fatal("injectable params must not be noise")
	}
}

func TestSelectParamsFiltersNoise(t *testing.T) {
	m := Materials{
		Params: []string{"id", "_gl", "cat", "playlist"},
		ParamScores: map[string]int{"id": 95, "_gl": 80, "cat": 70, "playlist": 75},
	}
	out := selectParamsForAssembly(m)
	for _, p := range out {
		if isNoiseParam(p) {
			t.Fatalf("noise param %q selected", p)
		}
	}
	if len(out) > MaxParamsAssemble {
		t.Fatalf("too many params: %d", len(out))
	}
}

func TestVolumeTypesForParamMaxTwo(t *testing.T) {
	idx := typesByID(AllDorkTypes())
	types := volumeTypesForParam("id", []string{"product", "index", "news"}, idx)
	if len(types) > MaxVolumeDorksPerParam {
		t.Fatalf("got %d volume types, want <= %d", len(types), MaxVolumeDorksPerParam)
	}
}
