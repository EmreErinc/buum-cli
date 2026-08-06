package exec

import (
	"context"
	"testing"
)

func TestTTYExecutorExitCodes(t *testing.T) {
	ex := TTYExecutor{}
	if r, _ := ex.Run(context.Background(), []string{"sh", "-c", "exit 0"}); r.ExitCode != 0 {
		t.Errorf("expected 0, got %d", r.ExitCode)
	}
	if r, _ := ex.Run(context.Background(), []string{"sh", "-c", "exit 3"}); r.ExitCode != 3 {
		t.Errorf("expected 3, got %d", r.ExitCode)
	}
}
