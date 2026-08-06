package ui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/emreerinc/buum/pkg/buum"
)

func TestJSONReportEvent(t *testing.T) {
	var w bytes.Buffer
	jw := &JSONWriter{W: &w}
	jw.ReportEvent(buum.Report{
		Results: []buum.ManagerResult{{Name: "brew", Status: buum.StatusOK}},
		Summary: buum.Summary{OK: 1},
	})
	var got map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(w.String())), &got); err != nil {
		t.Fatalf("not valid JSON: %v (%q)", err, w.String())
	}
	if got["event"] != "report" {
		t.Errorf("expected event=report, got %v", got["event"])
	}
}

func TestJSONPromptEvent(t *testing.T) {
	var w bytes.Buffer
	jw := &JSONWriter{W: &w}
	jw.Prompt("softwareupdate", []string{"softwareupdate", "-ia"}, "Password:")
	var got map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(w.String())), &got); err != nil {
		t.Fatal(err)
	}
	if got["event"] != "prompt" || got["manager"] != "softwareupdate" || got["text"] != "Password:" {
		t.Errorf("prompt event wrong: %v", got)
	}
}

func TestJSONList(t *testing.T) {
	var w bytes.Buffer
	jw := &JSONWriter{W: &w}
	jw.List(
		[]buum.Manager{{Name: "brew", DetectBin: "brew", NeedsSudo: false}},
		[]buum.Manager{{Name: "softwareupdate", DetectBin: "softwareupdate", NeedsSudo: true}},
	)
	var got map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(w.String())), &got); err != nil {
		t.Fatalf("not valid JSON: %v (%q)", err, w.String())
	}
	if _, ok := got["detected"]; !ok {
		t.Errorf("missing 'detected' key in JSON")
	}
	if _, ok := got["available"]; !ok {
		t.Errorf("missing 'available' key in JSON")
	}

	detected, ok := got["detected"].([]any)
	if !ok || len(detected) == 0 {
		t.Fatalf("detected is not an array or is empty: %v", got["detected"])
	}
	detectedEntry := detected[0].(map[string]any)
	if detectedEntry["name"] != "brew" {
		t.Errorf("detected entry name: got %v, want 'brew'", detectedEntry["name"])
	}
	if detectedEntry["detect_bin"] != "brew" {
		t.Errorf("detected entry detect_bin: got %v, want 'brew'", detectedEntry["detect_bin"])
	}

	available, ok := got["available"].([]any)
	if !ok || len(available) == 0 {
		t.Fatalf("available is not an array or is empty: %v", got["available"])
	}
	availableEntry := available[0].(map[string]any)
	if availableEntry["name"] != "softwareupdate" {
		t.Errorf("available entry name: got %v, want 'softwareupdate'", availableEntry["name"])
	}
	if availableEntry["needs_sudo"] != true {
		t.Errorf("available entry needs_sudo: got %v, want true", availableEntry["needs_sudo"])
	}
}
