package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/emreerinc/buum/pkg/buum"
	"gopkg.in/yaml.v3"
)

// fileManager is the YAML shape for one manager override/definition.
type fileManager struct {
	Enabled   *bool      `yaml:"enabled,omitempty"`
	DetectBin string     `yaml:"detect_bin,omitempty"`
	Steps     [][]string `yaml:"steps,omitempty"`
	NeedsSudo *bool      `yaml:"needs_sudo,omitempty"`
	Check     []string   `yaml:"check,omitempty"`
	Hint      string     `yaml:"hint,omitempty"`
}

type fileConfig struct {
	Order    []string               `yaml:"order,omitempty"`
	Managers map[string]fileManager `yaml:"managers,omitempty"`
}

// DefaultPath returns the standard config location.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "buum", "config.yaml")
}

// Load merges the YAML config at path over the built-in registry.
// A missing file yields pure built-ins with no error.
func Load(path string) (buum.Config, error) {
	base := buum.BuiltIns()

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return buum.Config{Managers: base}, nil
	}
	if err != nil {
		return buum.Config{Managers: base}, fmt.Errorf("reading config %s: %w", path, err)
	}

	var fc fileConfig
	if err := yaml.Unmarshal(raw, &fc); err != nil {
		// Parse error: report clearly, fall back to built-ins (do not hard-fail).
		return buum.Config{Managers: base}, fmt.Errorf("parsing config %s: %w", path, err)
	}

	idx := map[string]int{}
	for i, m := range base {
		idx[m.Name] = i
	}
	var customNames []string
	for name, fm := range fc.Managers {
		if i, ok := idx[name]; ok {
			apply(&base[i], fm)
		} else {
			customNames = append(customNames, name)
		}
	}
	sort.Strings(customNames)
	for _, name := range customNames {
		fm := fc.Managers[name]
		m := buum.Manager{Name: name, Enabled: true}
		apply(&m, fm)
		base = append(base, m)
		idx[name] = len(base) - 1
	}
	return buum.Config{Order: fc.Order, Managers: base}, nil
}

func apply(m *buum.Manager, fm fileManager) {
	if fm.Enabled != nil {
		m.Enabled = *fm.Enabled
	}
	if fm.DetectBin != "" {
		m.DetectBin = fm.DetectBin
	}
	if len(fm.Steps) > 0 {
		m.Steps = fm.Steps
	}
	if fm.NeedsSudo != nil {
		m.NeedsSudo = *fm.NeedsSudo
	}
	if len(fm.Check) > 0 {
		m.Check = fm.Check
		m.Precheck = nil // a user check replaces any built-in native precheck
	}
	if fm.Hint != "" {
		m.Hint = fm.Hint
	}
}

// Save merges an interactive selection into the config at path: each detected
// manager not in selected is written with enabled:false; reselected managers
// have the flag cleared (minimal diff). Content within the known schema (order,
// steps, custom managers) is preserved.
//
// Limitation: the file is round-tripped through the typed config schema, so
// comments AND any out-of-schema keys (unrecognized top-level or per-manager
// fields a user may have hand-added) are dropped on save. Editing the config by
// hand and using "save as default" are therefore best kept separate.
func Save(path string, detected []buum.Manager, selected []string) error {
	var fc fileConfig
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if uerr := yaml.Unmarshal(raw, &fc); uerr != nil {
			return fmt.Errorf("parsing config %s: %w", path, uerr)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("reading config %s: %w", path, err)
	}
	if fc.Managers == nil {
		fc.Managers = map[string]fileManager{}
	}

	sel := map[string]bool{}
	for _, n := range selected {
		sel[n] = true
	}
	for _, m := range detected {
		fm, existed := fc.Managers[m.Name]
		if sel[m.Name] {
			if !existed {
				continue // already enabled by default; no entry needed
			}
			fm.Enabled = nil // clear any disable
			fc.Managers[m.Name] = fm
		} else {
			no := false
			fm.Enabled = &no
			fc.Managers[m.Name] = fm
		}
	}

	out, err := yaml.Marshal(fc)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}
	return nil
}
