package exec

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestPTYExecutorCapturesPromptAndForwardsInput(t *testing.T) {
	var out bytes.Buffer
	var mu sync.Mutex
	var prompts []string

	// Model real interactive use: the answer is supplied only AFTER a prompt
	// is observed, never pre-staged. A pipe lets OnPrompt trigger the write.
	pr, pw := io.Pipe()
	var once sync.Once

	ex := &PTYExecutor{
		In:  pr,
		Out: &out,
		OnPrompt: func(step []string, text string) {
			mu.Lock()
			prompts = append(prompts, text)
			mu.Unlock()
			once.Do(func() {
				go func() {
					_, _ = pw.Write([]byte("secret\n"))
					_ = pw.Close()
				}()
			})
		},
		IdleMS: 150,
	}

	r, err := ex.Run(context.Background(), []string{"sh", "-c", `printf "Password: "; read x; echo "got:$x"`})
	if err != nil {
		t.Fatal(err)
	}
	if r.ExitCode != 0 {
		t.Fatalf("exit code %d", r.ExitCode)
	}
	if !strings.Contains(r.Output, "got:secret") {
		t.Errorf("forwarded input missing in output: %q", r.Output)
	}
	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, p := range prompts {
		if strings.Contains(p, "Password:") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a prompt event containing 'Password:', got %v", prompts)
	}
}
