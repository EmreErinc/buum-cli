package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSyncZshCompletionIdempotent(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".zcompdump-host-5.9"), []byte("x"), 0o644)
	root := &cobra.Command{Use: "buum"}

	for i := 0; i < 2; i++ {
		script, _, err := syncZshCompletion(root, home, home)
		if err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(script); !strings.Contains(string(b), "#compdef buum") {
			t.Fatalf("script missing compdef header: %s", script)
		}
	}
	rc, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if n := strings.Count(string(rc), "fpath=(~/.zsh/completions"); n != 1 {
		t.Fatalf("fpath block appended %d times, want 1", n)
	}
	if _, err := os.Stat(filepath.Join(home, ".zcompdump-host-5.9")); !os.IsNotExist(err) {
		t.Fatal("compdump should be removed")
	}
}

func TestHasCompletionFpathIgnoresComments(t *testing.T) {
	if hasCompletionFpath("# fpath=(~/.zsh/completions $fpath)") {
		t.Fatal("commented line must not count")
	}
	if !hasCompletionFpath("fpath=($HOME/.zsh/completions $fpath)") {
		t.Fatal("$HOME form should count")
	}
}
