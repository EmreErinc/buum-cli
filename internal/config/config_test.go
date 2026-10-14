package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emreerinc/buum/pkg/buum"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadMissingFileReturnsBuiltIns(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if len(cfg.Managers) == 0 {
		t.Fatal("expected built-ins when config absent")
	}
}

func TestLoadDisablesAndOverridesAndAddsCustom(t *testing.T) {
	cfg, err := Load(writeTemp(t, `
order: [brew, npm]
managers:
  pip:
    enabled: false
  brew:
    steps:
      - [brew, update]
  mytool:
    detect_bin: mytool
    steps:
      - [mytool, self-update]
`))
	if err != nil {
		t.Fatal(err)
	}
	byName := buumManager(cfg)
	if byName["pip"].Enabled {
		t.Error("pip should be disabled")
	}
	if got := byName["brew"].Steps; len(got) != 1 || got[0][1] != "update" {
		t.Errorf("brew steps not overridden: %+v", got)
	}
	if _, ok := byName["mytool"]; !ok {
		t.Error("custom manager mytool missing")
	}
	if len(cfg.Order) != 2 || cfg.Order[0] != "brew" {
		t.Errorf("order not parsed: %+v", cfg.Order)
	}
}

func TestLoadCheckOverridesBuiltInPrecheck(t *testing.T) {
	cfg, err := Load(writeTemp(t, `
managers:
  gem:
    check: [gem, environment, gemdir]
    hint: fix your ruby
`))
	if err != nil {
		t.Fatal(err)
	}
	gem := buumManager(cfg)["gem"]
	if len(gem.Check) != 3 || gem.Check[0] != "gem" {
		t.Errorf("check not parsed: %+v", gem.Check)
	}
	if gem.Hint != "fix your ruby" {
		t.Errorf("hint not parsed: %q", gem.Hint)
	}
	if gem.Precheck != nil {
		t.Error("user check must clear built-in precheck")
	}
}

func TestLoadParseErrorFallsBackToBuiltIns(t *testing.T) {
	cfg, err := Load(writeTemp(t, ":\n  - [oops\n bad: : :"))
	if err == nil {
		t.Fatal("expected parse error for malformed YAML")
	}
	if len(cfg.Managers) == 0 {
		t.Fatal("expected built-ins to be preserved on parse error")
	}
}

func buumManager(cfg buum.Config) map[string]buum.Manager {
	m := map[string]buum.Manager{}
	for _, x := range cfg.Managers {
		m[x.Name] = x
	}
	return m
}

func TestSaveDisablesDeselectedMinimalDiff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	detected := []buum.Manager{{Name: "brew"}, {Name: "npm"}, {Name: "cargo"}}
	selected := []string{"brew", "cargo"} // npm deselected

	if err := Save(path, detected, selected); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	by := buumManager(cfg)
	if by["npm"].Enabled {
		t.Error("npm should be disabled after Save")
	}
	if !by["brew"].Enabled || !by["cargo"].Enabled {
		t.Error("selected managers must stay enabled")
	}
}

func TestSaveReenablesAndPreservesOverrides(t *testing.T) {
	path := writeTemp(t, `
managers:
  npm:
    enabled: false
  brew:
    steps:
      - [brew, update]
`)
	detected := []buum.Manager{{Name: "brew"}, {Name: "npm"}}
	selected := []string{"brew", "npm"} // re-enable npm

	if err := Save(path, detected, selected); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	by := buumManager(cfg)
	if !by["npm"].Enabled {
		t.Error("npm should be re-enabled")
	}
	if got := by["brew"].Steps; len(got) != 1 || got[0][1] != "update" {
		t.Errorf("brew steps override must be preserved, got %+v", got)
	}
}

func TestSaveCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	if err := Save(path, []buum.Manager{{Name: "brew"}}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Save should create the file and parents: %v", err)
	}
}

func TestSavePreservesCommentsAndUnknownKeys(t *testing.T) {
	path := writeTemp(t, `# my buum config
order: [brew, npm] # brew first
managers:
  # keep pip off
  pip:
    enabled: false
  brew:
    steps:
      - [brew, update]
    my_note: hand-added
`)
	detected := []buum.Manager{{Name: "brew"}, {Name: "npm"}, {Name: "pip"}}
	if err := Save(path, detected, []string{"brew", "pip"}); err != nil { // npm off, pip back on
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	out := string(raw)
	for _, want := range []string{"# my buum config", "# brew first", "my_note: hand-added", "npm:\n    enabled: false"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "pip:") {
		t.Errorf("re-enabled pip entry should be removed:\n%s", out)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	by := buumManager(cfg)
	if by["npm"].Enabled || !by["pip"].Enabled || len(by["brew"].Steps) != 1 {
		t.Errorf("semantic result wrong: npm=%v pip=%v brew steps=%v", by["npm"].Enabled, by["pip"].Enabled, by["brew"].Steps)
	}
}

func TestSaveRefusesInvalidConfig(t *testing.T) {
	path := writeTemp(t, "managers: [not, a, map]\n")
	if err := Save(path, []buum.Manager{{Name: "brew"}}, nil); err == nil {
		t.Fatal("expected error for config Load would reject")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "managers: [not, a, map]\n" {
		t.Fatalf("invalid config must be left untouched, got %q", raw)
	}
}
