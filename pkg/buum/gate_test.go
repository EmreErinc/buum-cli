package buum

import (
	"context"
	"strings"
	"testing"
)

func withCheckRunner(t *testing.T, fn CheckRunner) {
	t.Helper()
	orig := checkRunner
	checkRunner = fn
	t.Cleanup(func() { checkRunner = orig })
}

func TestGatePrecheckSkipsWithReasonAndHint(t *testing.T) {
	m := Manager{
		Name:     "gem",
		Precheck: func() (bool, string) { return false, "gem dir not writable" },
		Hint:     "brew install ruby",
	}
	reason, skip := gate(context.Background(), m)
	if !skip {
		t.Fatal("expected skip")
	}
	if !strings.Contains(reason, "gem dir not writable") || !strings.Contains(reason, "brew install ruby") {
		t.Fatalf("reason missing detail/hint: %q", reason)
	}
}

func TestGatePrecheckPassesRuns(t *testing.T) {
	m := Manager{Name: "cargo", Precheck: func() (bool, string) { return true, "" }}
	if _, skip := gate(context.Background(), m); skip {
		t.Fatal("passing precheck must not skip")
	}
}

func TestGateNoCheckRuns(t *testing.T) {
	if _, skip := gate(context.Background(), Manager{Name: "brew"}); skip {
		t.Fatal("manager without checks must not skip")
	}
}

func TestGateCheckCommandOverridesPrecheck(t *testing.T) {
	// Precheck would pass, but the user Check command fails -> skip wins on Check.
	var got []string
	withCheckRunner(t, func(_ context.Context, cmd []string) bool {
		got = cmd
		return false
	})
	m := Manager{
		Name:     "gem",
		Check:    []string{"my", "check"},
		Precheck: func() (bool, string) { return true, "" },
		Hint:     "do the thing",
	}
	reason, skip := gate(context.Background(), m)
	if !skip {
		t.Fatal("failing check command must skip")
	}
	if strings.Join(got, " ") != "my check" {
		t.Fatalf("check runner got %v", got)
	}
	if !strings.Contains(reason, "do the thing") {
		t.Fatalf("reason missing hint: %q", reason)
	}
}

func TestGateCheckCommandPassesRuns(t *testing.T) {
	withCheckRunner(t, func(_ context.Context, _ []string) bool { return true })
	m := Manager{Name: "gem", Check: []string{"true"}}
	if _, skip := gate(context.Background(), m); skip {
		t.Fatal("passing check command must not skip")
	}
}

func TestRunSkipsGatedManagerButContinues(t *testing.T) {
	p := Plan{Managers: []Manager{
		{Name: "gem", Precheck: func() (bool, string) { return false, "no write" },
			Steps: [][]string{{"gem", "update"}}},
		{Name: "npm", Steps: [][]string{{"npm", "update", "-g"}}},
	}}
	ex := &fakeExec{codes: map[string]int{}}
	rep := Run(context.Background(), p, ex)

	if rep.Summary.Skipped != 1 || rep.Summary.OK != 1 || rep.Summary.Failed != 0 {
		t.Fatalf("summary wrong: %+v", rep.Summary)
	}
	for _, c := range ex.calls {
		if strings.HasPrefix(c, "gem") {
			t.Errorf("gated manager step must not run, calls=%v", ex.calls)
		}
	}
	if rep.Results[0].Status != StatusSkipped || rep.Results[0].SkipReason == "" {
		t.Errorf("gem result wrong: %+v", rep.Results[0])
	}
}
