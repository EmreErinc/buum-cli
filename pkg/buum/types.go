package buum

import "context"

// Manager describes one package manager and how to update it.
type Manager struct {
	Name      string     // "brew"
	DetectBin string     // binary that must be on PATH, e.g. "brew"
	Steps     [][]string // ordered commands, e.g. [["brew","update"],["brew","upgrade"]]
	NeedsSudo bool       // informational only (banner/list); does NOT cause a skip
	Enabled   bool

	// Check is a user-configured precondition command. A non-zero exit skips the
	// manager (Skipped, not Failed) before any step runs. When set it overrides
	// Precheck. Empty means no command check.
	Check []string
	// Precheck is a built-in native gate (e.g. filesystem writability) that cannot
	// be expressed as a simple command. Returns ok and a skip reason. A user Check
	// takes precedence. Not serialized.
	Precheck func() (ok bool, reason string) `json:"-" yaml:"-"`
	// Hint is remediation text appended to the skip reason when a check gates the
	// manager (e.g. "run: cargo install cargo-update").
	Hint string
}

// Config is the effective configuration after merging user config over built-ins.
type Config struct {
	Order    []string  // preferred run order by manager name; unlisted run after, in registry order
	Managers []Manager // full effective manager set
}

// Selection narrows which managers run. Only and Except are mutually exclusive.
// If Names is non-nil it is used verbatim (result of interactive selection).
type Selection struct {
	Only   []string
	Except []string
	Names  []string
}

// StepResult is the outcome of a single command.
type StepResult struct {
	Cmd      []string `json:"cmd"`
	ExitCode int      `json:"exit_code"`
	Output   string   `json:"output"` // captured output (PTY mode); empty in TTY mode
}

// Status is a manager's overall run status.
type Status string

const (
	StatusOK      Status = "ok"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
)

// ManagerResult is the outcome of running one manager's steps.
type ManagerResult struct {
	Name       string       `json:"name"`
	Status     Status       `json:"status"`
	ExitCode   int          `json:"exit_code"`
	SkipReason string       `json:"skip_reason,omitempty"`
	Steps      []StepResult `json:"steps"`
}

// Summary aggregates counts across a run.
type Summary struct {
	OK      int `json:"ok"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// Report is the full result of a Run.
type Report struct {
	Results []ManagerResult `json:"results"`
	Summary Summary         `json:"summary"`
}

// PathLookup reports whether a binary is found on PATH. Injectable for tests.
type PathLookup func(bin string) bool

// CheckRunner reports whether a precondition command succeeds (exit 0).
// Output is discarded. Injectable for tests.
type CheckRunner func(ctx context.Context, cmd []string) bool

// Executor runs a single command interactively and returns its result.
// Implementations own all IO/prompt behavior (TTY inherit, or PTY + prompt events).
type Executor interface {
	Run(ctx context.Context, cmd []string) (StepResult, error)
}
