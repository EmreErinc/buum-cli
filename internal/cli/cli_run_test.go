package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary compiles buum once into a temp dir and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "buum")
	out, err := exec.Command("go", "build", "-o", bin, "../../cmd/buum").CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func TestDryRunListsStepsExitZero(t *testing.T) {
	bin := buildBinary(t)
	// Use a config whose only manager detects on `sh` (always present) so the
	// plan is deterministic and no real package manager runs.
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
managers:
  fake:
    detect_bin: sh
    steps:
      - [sh, -c, "echo updating"]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "--config", cfgPath, "--only", "fake", "--dry-run")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry-run should exit 0: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "fake") || !strings.Contains(string(out), "echo updating") {
		t.Errorf("dry-run output missing plan: %q", out)
	}
}

func TestOnlyAndExceptConflictExit2(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--only", "brew", "--except", "npm")
	err := cmd.Run()
	ee, ok := err.(*exec.ExitError)
	if !ok || ee.ExitCode() != 2 {
		t.Fatalf("expected exit 2 on conflicting flags, got %v", err)
	}
}

func TestJSONRunEmitsOnlyJSONLines(t *testing.T) {
	bin := buildBinary(t)
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
managers:
  fake:
    detect_bin: sh
    steps:
      - [sh, -c, "echo updating"]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "--config", cfgPath, "--only", "fake", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("json run should exit 0: %v\n%s", err, out)
	}
	sawReport := false
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m map[string]any
		if uerr := json.Unmarshal([]byte(line), &m); uerr != nil {
			t.Fatalf("non-JSON line in --json output: %q (%v)", line, uerr)
		}
		if m["event"] == "report" {
			sawReport = true
		}
	}
	if !sawReport {
		t.Errorf("expected at least one report event, got: %q", out)
	}
}

func TestListJSONExitZero(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "list", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list --json should exit 0: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "detected") || !strings.Contains(string(out), "available") {
		t.Errorf("list --json missing keys: %q", out)
	}
}
