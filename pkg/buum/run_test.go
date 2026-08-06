package buum

import (
	"context"
	"strings"
	"testing"
)

// fakeExec returns a scripted exit code per command, keyed by the joined cmd.
type fakeExec struct {
	codes map[string]int
	calls []string
}

func (f *fakeExec) Run(_ context.Context, cmd []string) (StepResult, error) {
	key := strings.Join(cmd, " ")
	f.calls = append(f.calls, key)
	return StepResult{Cmd: cmd, ExitCode: f.codes[key]}, nil
}

func TestRunAllOK(t *testing.T) {
	p := Plan{Managers: []Manager{
		{Name: "brew", Steps: [][]string{{"brew", "update"}, {"brew", "upgrade"}}},
	}}
	ex := &fakeExec{codes: map[string]int{}}
	rep := Run(context.Background(), p, ex)
	if rep.Summary.OK != 1 || rep.Summary.Failed != 0 {
		t.Fatalf("summary wrong: %+v", rep.Summary)
	}
	if len(ex.calls) != 2 {
		t.Fatalf("expected 2 steps run, got %v", ex.calls)
	}
}

func TestRunFailedStepSkipsRemainingButContinuesNextManager(t *testing.T) {
	p := Plan{Managers: []Manager{
		{Name: "brew", Steps: [][]string{{"brew", "update"}, {"brew", "upgrade"}}},
		{Name: "npm", Steps: [][]string{{"npm", "update", "-g"}}},
	}}
	ex := &fakeExec{codes: map[string]int{"brew update": 1}} // first brew step fails
	rep := Run(context.Background(), p, ex)

	if rep.Summary.Failed != 1 || rep.Summary.OK != 1 {
		t.Fatalf("summary wrong: %+v", rep.Summary)
	}
	// brew upgrade must be skipped; npm must still run.
	for _, c := range ex.calls {
		if c == "brew upgrade" {
			t.Error("remaining step after failure must be skipped")
		}
	}
	if ex.calls[len(ex.calls)-1] != "npm update -g" {
		t.Errorf("next manager must still run, calls=%v", ex.calls)
	}
	if rep.Results[0].Status != StatusFailed || rep.Results[0].ExitCode != 1 {
		t.Errorf("brew result wrong: %+v", rep.Results[0])
	}
}
