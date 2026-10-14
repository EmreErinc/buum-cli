package cli

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/emreerinc/buum/pkg/buum"
)

func TestLastRunRoundTripReturnsOnlyFailed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buum", "last.json")
	if _, err := failedFromLastRun(path); err == nil {
		t.Fatal("missing last run must error")
	}
	rep := buum.Report{Results: []buum.ManagerResult{
		{Name: "brew", Status: buum.StatusOK},
		{Name: "npm", Status: buum.StatusFailed},
		{Name: "gem", Status: buum.StatusSkipped},
		{Name: "cargo", Status: buum.StatusFailed},
	}}
	if err := saveLastRun(path, rep); err != nil {
		t.Fatal(err)
	}
	got, err := failedFromLastRun(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"npm", "cargo"}) {
		t.Fatalf("got %v, want [npm cargo]", got)
	}
}

func TestLastRunPathHonorsXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/x/state")
	if got := lastRunPath(); got != "/x/state/buum/last.json" {
		t.Fatalf("got %s", got)
	}
}
