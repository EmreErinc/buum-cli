package buum

import "testing"

func fixtureCfg() Config {
	return Config{
		Order: []string{"npm", "brew"},
		Managers: []Manager{
			{Name: "brew", DetectBin: "brew", Enabled: true},
			{Name: "npm", DetectBin: "npm", Enabled: true},
			{Name: "mas", DetectBin: "mas", Enabled: true},
		},
	}
}

func allPresent(string) bool { return true }

func names(p Plan) []string {
	var out []string
	for _, m := range p.Managers {
		out = append(out, m.Name)
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildPlanAppliesOrder(t *testing.T) {
	p, err := BuildPlan(fixtureCfg(), Selection{}, allPresent)
	if err != nil {
		t.Fatal(err)
	}
	if !eq(names(p), []string{"npm", "brew", "mas"}) {
		t.Fatalf("order wrong: %v", names(p))
	}
}

func TestBuildPlanOnly(t *testing.T) {
	p, _ := BuildPlan(fixtureCfg(), Selection{Only: []string{"brew"}}, allPresent)
	if !eq(names(p), []string{"brew"}) {
		t.Fatalf("only failed: %v", names(p))
	}
}

func TestBuildPlanExcept(t *testing.T) {
	p, _ := BuildPlan(fixtureCfg(), Selection{Except: []string{"npm"}}, allPresent)
	if !eq(names(p), []string{"brew", "mas"}) {
		t.Fatalf("except failed: %v", names(p))
	}
}

func TestBuildPlanOnlyAndExceptConflict(t *testing.T) {
	_, err := BuildPlan(fixtureCfg(), Selection{Only: []string{"brew"}, Except: []string{"npm"}}, allPresent)
	if err == nil {
		t.Fatal("expected error when both Only and Except set")
	}
}

func TestBuildPlanExplicitNames(t *testing.T) {
	p, _ := BuildPlan(fixtureCfg(), Selection{Names: []string{"mas"}}, allPresent)
	if !eq(names(p), []string{"mas"}) {
		t.Fatalf("explicit names failed: %v", names(p))
	}
}

func TestBuildPlanEmptyNamesSelectsNothing(t *testing.T) {
	// A non-nil but empty Names is an explicit "select nothing" (e.g. the
	// interactive menu with every item deselected) — it must NOT fall through
	// to selecting everything.
	p, _ := BuildPlan(fixtureCfg(), Selection{Names: []string{}}, allPresent)
	if len(p.Managers) != 0 {
		t.Fatalf("empty non-nil Names must select nothing, got %v", names(p))
	}
}

func TestBuildPlanNilNamesIsNoFilter(t *testing.T) {
	// A nil Names means "no interactive filter" — all detected managers run.
	p, _ := BuildPlan(fixtureCfg(), Selection{Names: nil}, allPresent)
	if !eq(names(p), []string{"npm", "brew", "mas"}) {
		t.Fatalf("nil Names should keep all, got %v", names(p))
	}
}
