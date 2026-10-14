package config

import (
	"bytes"
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
// manager not in selected gets enabled:false; reselected managers have the flag
// removed (and their entry too, if nothing else is left in it).
//
// The file is edited as a YAML node tree, not re-encoded from the typed schema,
// so comments, key order and keys buum does not know survive. Indentation is
// normalized to 2 spaces. The write is atomic (temp file + rename).
func Save(path string, detected []buum.Manager, selected []string) error {
	var doc yaml.Node
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		// Typed decode only validates: refuse to edit a file Load would reject.
		var fc fileConfig
		if uerr := yaml.Unmarshal(raw, &fc); uerr != nil {
			return fmt.Errorf("parsing config %s: %w", path, uerr)
		}
		if uerr := yaml.Unmarshal(raw, &doc); uerr != nil {
			return fmt.Errorf("parsing config %s: %w", path, uerr)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("reading config %s: %w", path, err)
	}
	if doc.Kind == 0 { // missing or empty file
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{newMap()}}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("config %s: top level must be a mapping", path)
	}

	sel := map[string]bool{}
	for _, n := range selected {
		sel[n] = true
	}
	managers := mapGet(root, "managers")
	for _, m := range detected {
		if sel[m.Name] {
			entry := mapGet(managers, m.Name)
			if entry == nil {
				continue // already enabled by default; no entry needed
			}
			mapDelete(entry, "enabled")
			if len(entry.Content) == 0 {
				mapDelete(managers, m.Name)
			}
			continue
		}
		if managers == nil {
			managers = mapSet(root, "managers", newMap())
		}
		entry := mapGet(managers, m.Name)
		if entry == nil {
			entry = mapSet(managers, m.Name, newMap())
		}
		mapSet(entry, "enabled", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"})
	}
	if managers != nil && len(managers.Content) == 0 {
		mapDelete(root, "managers")
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("writing config %s: %w", path, err)
	}
	return nil
}

func newMap() *yaml.Node { return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"} }

// mapGet returns the value node for key in mapping m, or nil. A null value
// (e.g. "npm:" with nothing after it) is turned into an empty mapping in place
// so callers can add keys to it.
func mapGet(m *yaml.Node, key string) *yaml.Node {
	if m == nil {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			v := m.Content[i+1]
			if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
				*v = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", LineComment: v.LineComment}
			}
			return v
		}
	}
	return nil
}

// mapSet sets key to val in mapping m (replacing an existing value) and returns val.
func mapSet(m *yaml.Node, key string, val *yaml.Node) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = val
			return val
		}
	}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, val)
	return val
}

func mapDelete(m *yaml.Node, key string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}
