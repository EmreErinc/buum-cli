package buum

import "testing"

func TestBuiltInsHaveRequiredFields(t *testing.T) {
	got := BuiltIns()
	if len(got) == 0 {
		t.Fatal("BuiltIns() returned no managers")
	}
	names := map[string]bool{}
	for _, m := range got {
		if m.Name == "" || m.DetectBin == "" {
			t.Errorf("manager %+v missing Name or DetectBin", m)
		}
		if len(m.Steps) == 0 {
			t.Errorf("manager %q has no steps", m.Name)
		}
		for i, step := range m.Steps {
			if len(step) == 0 || step[0] == "" {
				t.Errorf("manager %q step %d is empty or has empty command", m.Name, i)
			}
		}
		if !m.Enabled {
			t.Errorf("built-in %q should default Enabled=true", m.Name)
		}
		names[m.Name] = true
	}
	for _, want := range []string{"brew", "npm", "mas", "pip", "softwareupdate"} {
		if !names[want] {
			t.Errorf("expected built-in manager %q", want)
		}
	}
}

func TestBuiltInsIncludeExpandedManagers(t *testing.T) {
	names := map[string]bool{}
	for _, m := range BuiltIns() {
		names[m.Name] = true
	}
	for _, want := range []string{
		"mise", "uv", "pnpm", "gcloud", "conda", "composer", "gh", "deno", "bun", "yarn",
		"port", "tlmgr",
		"flutter", "ghcup", "opam", "nix", "tldr",
	} {
		if !names[want] {
			t.Errorf("expected built-in manager %q", want)
		}
	}
}

func TestNixDetectsOnNixEnvBinary(t *testing.T) {
	for _, m := range BuiltIns() {
		if m.Name == "nix" && m.DetectBin != "nix-env" {
			t.Errorf("nix must detect on the nix-env binary, got %q", m.DetectBin)
		}
	}
}

func TestSudoManagersFlaggedAndPrefixed(t *testing.T) {
	sudoNames := map[string]bool{"softwareupdate": true, "port": true, "tlmgr": true}
	for _, m := range BuiltIns() {
		if !sudoNames[m.Name] {
			continue
		}
		if !m.NeedsSudo {
			t.Errorf("%q must set NeedsSudo=true", m.Name)
		}
		for i, step := range m.Steps {
			if step[0] != "sudo" {
				t.Errorf("%q step %d must begin with sudo, got %v", m.Name, i, step)
			}
		}
	}
}

func TestSoftwareupdateNeedsSudo(t *testing.T) {
	for _, m := range BuiltIns() {
		if m.Name == "softwareupdate" && !m.NeedsSudo {
			t.Error("softwareupdate must set NeedsSudo=true (informational)")
		}
	}
}
