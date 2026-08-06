package buum

import (
	"os"
	"os/exec"
	"strings"
)

// BuiltIns returns the curated default manager set with correct update commands.
// pip is report-only by design (no safe global "upgrade all"); users override in config.
func BuiltIns() []Manager {
	return []Manager{
		{Name: "brew", DetectBin: "brew", Enabled: true, Steps: [][]string{
			{"brew", "update"}, {"brew", "upgrade"}, {"brew", "cleanup"},
		}},
		{Name: "mas", DetectBin: "mas", Enabled: true, Steps: [][]string{
			{"mas", "upgrade"},
		}},
		{Name: "npm", DetectBin: "npm", Enabled: true, Steps: [][]string{
			{"npm", "update", "-g"},
		}},
		{Name: "pip", DetectBin: "pip3", Enabled: true, Steps: [][]string{
			{"pip3", "list", "--outdated"},
		}},
		{Name: "gem", DetectBin: "gem", Enabled: true, Steps: [][]string{
			{"gem", "update"},
		}, Precheck: gemGemdirWritable,
			Hint: "system Ruby is not writable; use a non-system Ruby (brew install ruby) " +
				"or override gem steps in config (e.g. [gem, update, --user-install])"},
		{Name: "pipx", DetectBin: "pipx", Enabled: true, Steps: [][]string{
			{"pipx", "upgrade-all"},
		}},
		{Name: "cargo", DetectBin: "cargo", Enabled: true, Steps: [][]string{
			{"cargo", "install-update", "-a"},
		}, Precheck: hasCargoInstallUpdate,
			Hint: "run: cargo install cargo-update"},
		{Name: "rustup", DetectBin: "rustup", Enabled: true, Steps: [][]string{
			{"rustup", "update"},
		}},
		{Name: "softwareupdate", DetectBin: "softwareupdate", Enabled: true, NeedsSudo: true, Steps: [][]string{
			{"sudo", "softwareupdate", "-ia"},
		}},

		// --- Expanded set: additional developer tools with a clean, detectable
		// "upgrade everything installed" command. Uninstalled tools are hidden
		// by PATH detection. See docs/superpowers/specs/2026-07-08-expand-manager-registry-design.md

		{Name: "mise", DetectBin: "mise", Enabled: true, Steps: [][]string{
			{"mise", "upgrade"},
		}},
		{Name: "uv", DetectBin: "uv", Enabled: true, Steps: [][]string{
			{"uv", "tool", "upgrade", "--all"},
		}},
		{Name: "pnpm", DetectBin: "pnpm", Enabled: true, Steps: [][]string{
			{"pnpm", "update", "-g"},
		}},
		{Name: "gcloud", DetectBin: "gcloud", Enabled: true, Steps: [][]string{
			{"gcloud", "components", "update", "-q"},
		}},
		{Name: "conda", DetectBin: "conda", Enabled: true, Steps: [][]string{
			{"conda", "update", "--all", "-y"},
		}},
		{Name: "composer", DetectBin: "composer", Enabled: true, Steps: [][]string{
			{"composer", "self-update"},
			{"composer", "global", "update"},
		}},
		{Name: "gh", DetectBin: "gh", Enabled: true, Steps: [][]string{
			{"gh", "extension", "upgrade", "--all"},
		}},
		{Name: "deno", DetectBin: "deno", Enabled: true, Steps: [][]string{
			{"deno", "upgrade"},
		}},
		{Name: "bun", DetectBin: "bun", Enabled: true, Steps: [][]string{
			{"bun", "upgrade"},
		}},
		{Name: "yarn", DetectBin: "yarn", Enabled: true, Steps: [][]string{
			{"yarn", "global", "upgrade"},
		}},

		{Name: "port", DetectBin: "port", Enabled: true, NeedsSudo: true, Steps: [][]string{
			{"sudo", "port", "selfupdate"},
			{"sudo", "port", "upgrade", "outdated"},
		}},
		{Name: "tlmgr", DetectBin: "tlmgr", Enabled: true, NeedsSudo: true, Steps: [][]string{
			{"sudo", "tlmgr", "update", "--self", "--all"},
		}},

		{Name: "flutter", DetectBin: "flutter", Enabled: true, Steps: [][]string{
			{"flutter", "upgrade"},
		}},
		{Name: "ghcup", DetectBin: "ghcup", Enabled: true, Steps: [][]string{
			{"ghcup", "upgrade"},
		}},
		{Name: "opam", DetectBin: "opam", Enabled: true, Steps: [][]string{
			{"opam", "update"},
			{"opam", "upgrade", "-y"},
		}},
		{Name: "nix", DetectBin: "nix-env", Enabled: true, Steps: [][]string{
			{"nix-channel", "--update"},
			{"nix-env", "-u"},
		}},
		{Name: "tldr", DetectBin: "tldr", Enabled: true, Steps: [][]string{
			{"tldr", "--update"},
		}},
	}
}

// gemGemdirWritable reports whether `gem update` can write to the active gem dir.
// System Ruby (e.g. /Library/Ruby/Gems/2.6.0) is not user-writable and fails
// mid-update; catch it up front. If the dir cannot be determined, do not block —
// let the real step decide.
func gemGemdirWritable() (bool, string) {
	out, err := exec.Command("gem", "environment", "gemdir").Output()
	if err != nil {
		return true, ""
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return true, ""
	}
	f, err := os.CreateTemp(dir, ".buum-write-*")
	if err != nil {
		return false, "gem dir " + dir + " is not writable"
	}
	f.Close()
	os.Remove(f.Name())
	return true, ""
}

// hasCargoInstallUpdate reports whether the `cargo install-update` subcommand is
// available. External cargo subcommands resolve to a `cargo-<name>` binary on
// PATH; its absence is what produces "no such command: install-update".
func hasCargoInstallUpdate() (bool, string) {
	if _, err := exec.LookPath("cargo-install-update"); err == nil {
		return true, ""
	}
	return false, "cargo-update not installed (cargo install-update subcommand missing)"
}
