package buum

import "testing"

func TestDetectFindsOnlyPresentEnabled(t *testing.T) {
	cfg := Config{Managers: []Manager{
		{Name: "brew", DetectBin: "brew", Enabled: true},
		{Name: "npm", DetectBin: "npm", Enabled: true},
		{Name: "mas", DetectBin: "mas", Enabled: false}, // disabled
	}}
	present := map[string]bool{"brew": true, "mas": true}
	look := func(bin string) bool { return present[bin] }

	got := Detect(cfg, look)

	if len(got) != 1 || got[0].Name != "brew" {
		t.Fatalf("expected only brew detected, got %+v", got)
	}
}
