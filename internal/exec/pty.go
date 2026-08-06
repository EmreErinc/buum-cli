package exec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/emreerinc/buum/pkg/buum"
)

// PTYExecutor runs a command under a pseudo-terminal so output can be captured
// while the child stays interactive. When output goes idle mid-line (no trailing
// newline), it treats the trailing text as a prompt and calls OnPrompt; answers
// are read from In and forwarded to the child. It never auto-answers or times out.
type PTYExecutor struct {
	In       io.Reader                        // answer source (default os.Stdin)
	Out      io.Writer                        // mirror of child output (default os.Stdout)
	OnPrompt func(step []string, text string) // called with the current step and pending prompt text
	IdleMS   int                              // idle window to treat a partial line as a prompt (default 300)
}

func (e *PTYExecutor) Run(ctx context.Context, cmd []string) (buum.StepResult, error) {
	in := e.In
	if in == nil {
		in = os.Stdin
	}
	out := e.Out
	if out == nil {
		out = os.Stdout
	}
	idle := e.IdleMS
	if idle <= 0 {
		idle = 300
	}

	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	f, err := pty.Start(c)
	if err != nil {
		return buum.StepResult{Cmd: cmd}, err
	}
	defer f.Close()

	// Forward answers from In into the pty. Input flows to the child exactly
	// as the driver supplies it: a human typing at os.Stdin, or a JSON-mode
	// controller writing an answer after it sees a prompt event. buum never
	// synthesizes input and never times out waiting for it.
	//
	// Known limitation: when In is os.Stdin (an unbounded reader) this
	// goroutine can stay blocked on Read after the child exits. For buum's
	// sequential, short-lived execution that is acceptable; a bounded In (as
	// tests use) returns EOF and lets the goroutine exit.
	go func() { _, _ = io.Copy(f, in) }()

	var captured bytes.Buffer
	var mu sync.Mutex
	var line strings.Builder // current unterminated line
	done := false            // set once Run is returning; guards late OnPrompt

	var timer *time.Timer
	arm := func() {
		if e.OnPrompt == nil {
			return
		}
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(time.Duration(idle)*time.Millisecond, func() {
			mu.Lock()
			pending := line.String()
			finished := done
			mu.Unlock()
			if !finished && strings.TrimSpace(pending) != "" {
				e.OnPrompt(cmd, pending)
			}
		})
	}

	buf := make([]byte, 4096)
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			_, _ = out.Write(chunk)
			captured.Write(chunk)

			mu.Lock()
			for _, b := range chunk {
				if b == '\n' {
					line.Reset()
				} else {
					line.WriteByte(b)
				}
			}
			mu.Unlock()
			arm()
		}
		if rerr != nil {
			break // includes EIO when the child exits (normal on macOS)
		}
	}
	mu.Lock()
	done = true
	mu.Unlock()
	if timer != nil {
		timer.Stop()
	}

	werr := c.Wait()
	code := 0
	if werr != nil {
		var ee *exec.ExitError
		if errors.As(werr, &ee) {
			code = ee.ExitCode()
			werr = nil
		}
	}
	return buum.StepResult{Cmd: cmd, ExitCode: code, Output: captured.String()}, werr
}
