package buum

import (
	"context"
	"os/exec"
	"strings"
)

// DefaultCheck runs a precondition command and reports success (exit 0),
// discarding all output.
var DefaultCheck CheckRunner = func(ctx context.Context, cmd []string) bool {
	if len(cmd) == 0 {
		return true
	}
	return exec.CommandContext(ctx, cmd[0], cmd[1:]...).Run() == nil
}

// checkRunner evaluates user Check commands; swappable in tests.
var checkRunner = DefaultCheck

// gate reports whether m should be skipped before its steps run, with a reason.
// A user-configured Check command takes precedence over a built-in Precheck.
func gate(ctx context.Context, m Manager) (reason string, skip bool) {
	switch {
	case len(m.Check) > 0:
		if checkRunner(ctx, m.Check) {
			return "", false
		}
		return withHint("precheck failed: "+strings.Join(m.Check, " "), m.Hint), true
	case m.Precheck != nil:
		if ok, r := m.Precheck(); !ok {
			return withHint(r, m.Hint), true
		}
	}
	return "", false
}

func withHint(reason, hint string) string {
	if hint == "" {
		return reason
	}
	return reason + " — hint: " + hint
}
