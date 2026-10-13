package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// zshRCBlock is appended to .zshrc only when no fpath line already covers
// ~/.zsh/completions. compinit is repeated because the block lands after any
// framework (e.g. oh-my-zsh) that already ran it.
const zshRCBlock = `
# buum shell completion
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit && compinit
`

// addZshSyncCmd attaches "sync" under cobra's default "completion zsh" command.
func addZshSyncCmd(root *cobra.Command) {
	root.InitDefaultCompletionCmd()
	zsh, _, err := root.Find([]string{"completion", "zsh"})
	if err != nil || zsh.Name() != "zsh" {
		return
	}
	zsh.AddCommand(&cobra.Command{
		Use:   "sync",
		Short: "Install/refresh the zsh completion script and wire it into ~/.zshrc",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			zdot := os.Getenv("ZDOTDIR")
			if zdot == "" {
				zdot = home
			}
			script, rcChanged, err := syncZshCompletion(root, home, zdot)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "wrote", script)
			if rcChanged {
				fmt.Fprintln(out, "added completion fpath to", filepath.Join(zdot, ".zshrc"))
			}
			fmt.Fprintln(out, "restart your shell (exec zsh) to load it")
			return nil
		},
	})
}

// syncZshCompletion writes _buum into home/.zsh/completions, appends zshRCBlock
// to zdot/.zshrc if needed, and deletes zdot/.zcompdump* so compinit rebuilds.
func syncZshCompletion(root *cobra.Command, home, zdot string) (script string, rcChanged bool, err error) {
	var buf bytes.Buffer
	if err := root.GenZshCompletion(&buf); err != nil {
		return "", false, fmt.Errorf("generating zsh completion: %w", err)
	}
	dir := filepath.Join(home, ".zsh", "completions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, fmt.Errorf("creating %s: %w", dir, err)
	}
	script = filepath.Join(dir, "_buum")
	if err := os.WriteFile(script, buf.Bytes(), 0o644); err != nil {
		return "", false, fmt.Errorf("writing %s: %w", script, err)
	}

	rc := filepath.Join(zdot, ".zshrc")
	raw, err := os.ReadFile(rc)
	if err != nil && !os.IsNotExist(err) {
		return script, false, fmt.Errorf("reading %s: %w", rc, err)
	}
	if !hasCompletionFpath(string(raw)) {
		f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return script, false, fmt.Errorf("opening %s: %w", rc, err)
		}
		_, werr := f.WriteString(zshRCBlock)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return script, false, fmt.Errorf("writing %s: %w", rc, werr)
		}
		rcChanged = true
	}

	// Stale dumps can keep an old completion mapping; zsh regenerates them.
	dumps, _ := filepath.Glob(filepath.Join(zdot, ".zcompdump*"))
	for _, d := range dumps {
		os.Remove(d)
	}
	return script, rcChanged, nil
}

func hasCompletionFpath(rc string) bool {
	for _, line := range strings.Split(rc, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "fpath") && strings.Contains(line, ".zsh/completions") {
			return true
		}
	}
	return false
}
