package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emreerinc/buum/pkg/buum"
)

// lastRunPath is where the report of the most recent real run is kept:
// $XDG_STATE_HOME/buum/last.json, defaulting to ~/.local/state/buum/last.json.
func lastRunPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "buum", "last.json")
}

// saveLastRun writes rep atomically (temp file + rename).
func saveLastRun(path string, rep buum.Report) error {
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// failedFromLastRun returns the managers that failed in the saved report.
// Skipped managers are not included: their precondition decided, not a failure.
func failedFromLastRun(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no previous run recorded (%s)", path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading last run: %w", err)
	}
	var rep buum.Report
	if err := json.Unmarshal(b, &rep); err != nil {
		return nil, fmt.Errorf("parsing last run %s: %w", path, err)
	}
	failed := []string{}
	for _, r := range rep.Results {
		if r.Status == buum.StatusFailed {
			failed = append(failed, r.Name)
		}
	}
	return failed, nil
}
