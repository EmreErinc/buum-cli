package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/emreerinc/buum/pkg/buum"
)

func TestSummaryShowsCountsAndStatuses(t *testing.T) {
	rep := buum.Report{
		Results: []buum.ManagerResult{
			{Name: "brew", Status: buum.StatusOK},
			{Name: "npm", Status: buum.StatusFailed, ExitCode: 1},
		},
		Summary: buum.Summary{OK: 1, Failed: 1},
	}
	var w bytes.Buffer
	Summary(&w, rep)
	s := w.String()
	if !strings.Contains(s, "brew") || !strings.Contains(s, "ok") {
		t.Errorf("missing brew/ok: %q", s)
	}
	if !strings.Contains(s, "npm") || !strings.Contains(s, "failed") {
		t.Errorf("missing npm/failed: %q", s)
	}
	if !strings.Contains(s, "1 ok") || !strings.Contains(s, "1 failed") {
		t.Errorf("missing totals: %q", s)
	}
}

func TestSelectParsesIndices(t *testing.T) {
	managers := []buum.Manager{{Name: "brew"}, {Name: "npm"}, {Name: "mas"}}
	got, err := selectNumbered(strings.NewReader("1,3\n"), &bytes.Buffer{}, managers)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "brew" || got[1] != "mas" {
		t.Fatalf("select wrong: %v", got)
	}
}

func TestSelectEmptyMeansAll(t *testing.T) {
	managers := []buum.Manager{{Name: "brew"}, {Name: "npm"}}
	got, _ := selectNumbered(strings.NewReader("\n"), &bytes.Buffer{}, managers)
	if len(got) != 2 {
		t.Fatalf("empty should select all, got %v", got)
	}
}
