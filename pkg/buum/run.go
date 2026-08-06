package buum

import "context"

// Run executes the plan sequentially. Before a manager's steps run, its precheck
// (user Check command or built-in Precheck) is evaluated; a failing check marks
// the manager Skipped (with a reason) and no steps run. Each step runs via ex. A
// non-zero step exit marks the manager failed and skips its remaining steps;
// execution then continues with the next manager. buum never auto-skips based on
// NeedsSudo.
func Run(ctx context.Context, p Plan, ex Executor) Report {
	var rep Report
	for _, m := range p.Managers {
		if reason, skip := gate(ctx, m); skip {
			rep.Results = append(rep.Results, ManagerResult{
				Name: m.Name, Status: StatusSkipped, SkipReason: reason,
			})
			rep.Summary.Skipped++
			continue
		}
		res := ManagerResult{Name: m.Name, Status: StatusOK}
		for _, step := range m.Steps {
			sr, err := ex.Run(ctx, step)
			res.Steps = append(res.Steps, sr)
			if err != nil || sr.ExitCode != 0 {
				res.Status = StatusFailed
				if sr.ExitCode != 0 {
					res.ExitCode = sr.ExitCode
				} else {
					res.ExitCode = 1
				}
				break // skip remaining steps for this manager
			}
		}
		switch res.Status {
		case StatusOK:
			rep.Summary.OK++
		case StatusFailed:
			rep.Summary.Failed++
		case StatusSkipped:
			rep.Summary.Skipped++
		}
		rep.Results = append(rep.Results, res)
	}
	return rep
}
