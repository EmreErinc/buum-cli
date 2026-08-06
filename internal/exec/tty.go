package exec

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/emreerinc/buum/pkg/buum"
)

// TTYExecutor runs a command with the child inheriting buum's controlling
// terminal, so interactive prompts appear live and block for real input.
type TTYExecutor struct{}

func (TTYExecutor) Run(ctx context.Context, cmd []string) (buum.StepResult, error) {
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err := c.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
			err = nil // non-zero exit is a result, not an executor error
		}
	}
	return buum.StepResult{Cmd: cmd, ExitCode: code}, err
}
