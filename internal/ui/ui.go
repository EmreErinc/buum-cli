package ui

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/emreerinc/buum/pkg/buum"
)

// Banner announces the manager about to run.
func Banner(w io.Writer, name string) {
	fmt.Fprintf(w, "→ %s\n", name)
}

// Summary prints a per-manager status table and totals.
func Summary(w io.Writer, rep buum.Report) {
	fmt.Fprintln(w, "\nSummary:")
	for _, r := range rep.Results {
		line := fmt.Sprintf("  %-16s %s", r.Name, r.Status)
		if r.Status == buum.StatusFailed {
			line += fmt.Sprintf(" (exit %d)", r.ExitCode)
		}
		if r.SkipReason != "" {
			line += fmt.Sprintf(" (%s)", r.SkipReason)
		}
		fmt.Fprintln(w, line)
	}
	fmt.Fprintf(w, "  %d ok, %d failed, %d skipped\n",
		rep.Summary.OK, rep.Summary.Failed, rep.Summary.Skipped)
}

// selectNumbered is the non-TTY fallback: a numbered list read as a
// comma-separated line of 1-based indices. Empty or "all" selects everything.
func selectNumbered(in io.Reader, w io.Writer, managers []buum.Manager) ([]string, error) {
	fmt.Fprintln(w, "Select managers to update (comma-separated numbers, empty = all):")
	for i, m := range managers {
		fmt.Fprintf(w, "  %d) %s\n", i+1, m.Name)
	}
	fmt.Fprint(w, "> ")

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" || strings.EqualFold(line, "all") {
		var all []string
		for _, m := range managers {
			all = append(all, m.Name)
		}
		return all, nil
	}

	var out []string
	for _, tok := range strings.Split(line, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		n, err := strconv.Atoi(tok)
		if err != nil || n < 1 || n > len(managers) {
			return nil, fmt.Errorf("invalid selection %q", tok)
		}
		out = append(out, managers[n-1].Name)
	}
	return out, nil
}
