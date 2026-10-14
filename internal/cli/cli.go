package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/emreerinc/buum/internal/config"
	uiexec "github.com/emreerinc/buum/internal/exec"
	"github.com/emreerinc/buum/internal/ui"
	"github.com/emreerinc/buum/pkg/buum"
)

// Version is the buum build version. Releases set it from the git tag via
// GoReleaser ldflags (-X .../internal/cli.Version=...); plain go builds say "dev".
var Version = "dev"

type opts struct {
	only        []string
	except      []string
	dryRun      bool
	interactive bool
	jsonOut     bool
	verbose     bool
	configPath  string
	retryFailed bool
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	o := &opts{}
	root := &cobra.Command{
		Use:           "buum",
		Short:         "Detect installed package managers and run their updates",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, o)
		},
	}
	f := root.PersistentFlags()
	f.StringSliceVar(&o.only, "only", nil, "run only these managers (comma-separated)")
	f.StringSliceVar(&o.except, "except", nil, "run all detected except these")
	f.BoolVar(&o.dryRun, "dry-run", false, "print steps that would run; execute nothing")
	f.BoolVarP(&o.interactive, "interactive", "i", false, "pick managers interactively before running")
	f.BoolVar(&o.jsonOut, "json", false, "machine-readable JSON event output")
	f.BoolVarP(&o.verbose, "verbose", "v", false, "extra logging")
	f.StringVar(&o.configPath, "config", config.DefaultPath(), "path to config file")
	root.Flags().BoolVar(&o.retryFailed, "retry-failed", false, "rerun only the managers that failed in the last run")

	root.AddCommand(listCmd(o), configCmd(o), versionCmd())
	addZshSyncCmd(root)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return exitCodeFor(err)
	}
	return lastExit
}

// lastExit carries the run's exit code (0/1) out of RunE, since a failed
// manager is not a Go error. Usage/config errors return via error → exit 2.
var lastExit int

type usageError struct{ err error }

func (u usageError) Error() string { return u.err.Error() }

func exitCodeFor(err error) int {
	if _, ok := err.(usageError); ok {
		return 2
	}
	return 2 // all cobra-surfaced errors are usage/config problems
}

func loadPlan(o *opts) (buum.Config, buum.Plan, error) {
	cfg, err := config.Load(o.configPath)
	if err != nil {
		// Parse error already fell back to built-ins inside Load; surface note.
		fmt.Fprintln(os.Stderr, "warning:", err)
	}
	sel := buum.Selection{Only: o.only, Except: o.except}
	if o.retryFailed {
		if len(o.only) > 0 || len(o.except) > 0 || o.interactive {
			return cfg, buum.Plan{}, usageError{errors.New("--retry-failed cannot be combined with --only, --except or -i")}
		}
		names, ferr := failedFromLastRun(lastRunPath())
		if ferr != nil {
			return cfg, buum.Plan{}, usageError{ferr}
		}
		sel = buum.Selection{Names: names} // non-nil: empty means run nothing
	}
	plan, perr := buum.BuildPlan(cfg, sel, buum.DefaultLookup)
	if perr != nil {
		return cfg, buum.Plan{}, usageError{perr}
	}
	return cfg, plan, nil
}

func runUpdate(cmd *cobra.Command, o *opts) error {
	cfg, plan, err := loadPlan(o)
	if err != nil {
		return err
	}
	if o.retryFailed && len(plan.Managers) == 0 {
		fmt.Fprintln(os.Stderr, "nothing to retry: no failed managers in the last run")
		lastExit = 0
		return nil
	}

	if o.interactive && !o.dryRun {
		detected := plan.Managers
		names, serr := ui.SelectManagers(os.Stdin, os.Stdout, plan.Managers, nil)
		if errors.Is(serr, ui.ErrSelectCanceled) {
			fmt.Fprintln(os.Stderr, "canceled")
			lastExit = 0
			return nil
		}
		if serr != nil {
			return usageError{serr}
		}
		// Feed the interactive result back through the planner via the
		// Selection.Names seam (non-nil = verbatim; empty = run nothing).
		plan, err = buum.BuildPlan(cfg, buum.Selection{Names: names}, buum.DefaultLookup)
		if err != nil {
			return usageError{err}
		}

		if term.IsTerminal(int(os.Stdin.Fd())) && ui.PromptYesNo(os.Stdin, os.Stdout, "Save this selection as default? [y/N] ") {
			if serr := config.Save(o.configPath, detected, names); serr != nil {
				fmt.Fprintln(os.Stderr, "warning: save failed:", serr)
			}
		}
	}

	if !o.dryRun {
		defer primeSudo(plan)()
	}

	if o.dryRun {
		for _, m := range plan.Managers {
			fmt.Fprintf(os.Stdout, "→ %s\n", m.Name)
			for _, step := range m.Steps {
				fmt.Fprintf(os.Stdout, "    %s\n", strings.Join(step, " "))
			}
		}
		lastExit = 0
		return nil
	}

	var rep buum.Report
	if o.jsonOut {
		jw := &ui.JSONWriter{W: os.Stdout}
		rep = runJSON(cmd.Context(), plan, jw)
		jw.ReportEvent(rep)
	} else {
		rep = runTTY(cmd.Context(), plan)
		ui.Summary(os.Stdout, rep)
	}
	if serr := saveLastRun(lastRunPath(), rep); serr != nil {
		fmt.Fprintln(os.Stderr, "warning: could not save run report:", serr)
	}

	if rep.Summary.Failed > 0 {
		lastExit = 1
	} else {
		lastExit = 0
	}
	return nil
}

// needsSudo reports whether any step in the plan runs through sudo (built-in or
// user-configured).
func needsSudo(plan buum.Plan) bool {
	for _, m := range plan.Managers {
		for _, step := range m.Steps {
			if len(step) > 0 && step[0] == "sudo" {
				return true
			}
		}
	}
	return false
}

// primeSudo asks for the sudo password once before any manager runs and keeps
// the sudo timestamp fresh until the returned stop is called, so sudo steps late
// in a long run do not sit waiting for a password. Best effort: on failure the
// sudo steps simply prompt as before.
func primeSudo(plan buum.Plan) (stop func()) {
	if !needsSudo(plan) || !term.IsTerminal(int(os.Stdin.Fd())) {
		return func() {}
	}
	fmt.Fprintln(os.Stderr, "Some managers need sudo; asking for your password once up front.")
	v := exec.Command("sudo", "-v")
	v.Stdin, v.Stdout, v.Stderr = os.Stdin, os.Stderr, os.Stderr
	if err := v.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: sudo not primed; sudo steps will ask again:", err)
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				exec.Command("sudo", "-n", "-v").Run()
			}
		}
	}()
	return func() { close(done) }
}

func runTTY(ctx context.Context, plan buum.Plan) buum.Report {
	ex := uiexec.TTYExecutor{}
	// Banner per manager: wrap the plan run manager-by-manager.
	var rep buum.Report
	for _, m := range plan.Managers {
		ui.Banner(os.Stderr, m.Name)
		sub := buum.Run(ctx, buum.Plan{Managers: []buum.Manager{m}}, ex)
		if len(sub.Results) == 1 && sub.Results[0].Status == buum.StatusSkipped {
			fmt.Fprintf(os.Stderr, "  skipped: %s\n", sub.Results[0].SkipReason)
		}
		mergeReport(&rep, sub)
	}
	return rep
}

func runJSON(ctx context.Context, plan buum.Plan, jw *ui.JSONWriter) buum.Report {
	var rep buum.Report
	for _, m := range plan.Managers {
		mName := m.Name
		ex := &uiexec.PTYExecutor{
			Out:      io.Discard,
			OnPrompt: func(step []string, text string) { jw.Prompt(mName, step, text) },
		}
		sub := buum.Run(ctx, buum.Plan{Managers: []buum.Manager{m}}, ex)
		mergeReport(&rep, sub)
	}
	return rep
}

func mergeReport(dst *buum.Report, src buum.Report) {
	dst.Results = append(dst.Results, src.Results...)
	dst.Summary.OK += src.Summary.OK
	dst.Summary.Failed += src.Summary.Failed
	dst.Summary.Skipped += src.Summary.Skipped
}

func listCmd(o *opts) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show detected and known-but-missing managers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(o.configPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, "warning:", err)
			}
			detected := buum.Detect(cfg, buum.DefaultLookup)
			detectedNames := map[string]bool{}
			for _, m := range detected {
				detectedNames[m.Name] = true
			}
			var available []buum.Manager
			for _, m := range cfg.Managers {
				if m.Enabled && !detectedNames[m.Name] {
					available = append(available, m)
				}
			}
			if o.jsonOut {
				(&ui.JSONWriter{W: os.Stdout}).List(detected, available)
			} else {
				fmt.Println("Detected:")
				for _, m := range detected {
					fmt.Printf("  %s\n", m.Name)
				}
				fmt.Println("Available (not installed):")
				for _, m := range available {
					fmt.Printf("  %s\n", m.Name)
				}
			}
			lastExit = 0
			return nil
		},
	}
}

func configCmd(o *opts) *cobra.Command {
	c := &cobra.Command{Use: "config", Short: "Config file helpers"}
	c.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print config file path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(o.configPath)
		},
	})
	c.AddCommand(&cobra.Command{
		Use:   "edit",
		Short: "Open config file in $EDITOR",
		RunE: func(cmd *cobra.Command, args []string) error {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			ed := exec.Command(editor, o.configPath)
			ed.Stdin, ed.Stdout, ed.Stderr = os.Stdin, os.Stdout, os.Stderr
			return ed.Run()
		},
	})
	c.AddCommand(&cobra.Command{
		Use:   "set",
		Short: "Toggle which installed managers are enabled",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(o)
		},
	})
	return c
}

// runConfigSet shows every known manager (enabled or not) as a checkbox list
// seeded from the config, installed ones first, and saves the enabled set.
func runConfigSet(o *opts) error {
	// No numbered fallback: its "empty = all" would silently enable everything.
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return usageError{errors.New("config set requires a terminal; use 'buum config edit'")}
	}
	cfg, err := config.Load(o.configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning:", err)
	}
	var installed, missing []buum.Manager
	sections := map[string]string{}
	for _, m := range cfg.Managers {
		if buum.DefaultLookup(m.DetectBin) {
			installed = append(installed, m)
			sections[m.Name] = "Installed"
		} else {
			missing = append(missing, m)
			sections[m.Name] = "Not installed"
		}
	}
	all := append(installed, missing...)
	names, err := ui.SelectManagers(os.Stdin, os.Stdout, all, sections)
	if errors.Is(err, ui.ErrSelectCanceled) {
		fmt.Fprintln(os.Stderr, "canceled")
		return nil
	}
	if err != nil {
		return usageError{err}
	}
	if err := config.Save(o.configPath, all, names); err != nil {
		return err
	}
	fmt.Printf("saved %d of %d enabled → %s\n", len(names), len(all), o.configPath)
	return nil
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print buum version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), Version)
		},
	}
}
