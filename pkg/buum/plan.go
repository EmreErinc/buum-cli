package buum

import "errors"

// Plan is the ordered set of managers to run.
type Plan struct {
	Managers []Manager
}

// ErrSelectionConflict is returned when both Only and Except are supplied.
var ErrSelectionConflict = errors.New("--only and --except are mutually exclusive")

// BuildPlan detects active managers, applies Order, then applies the selection.
func BuildPlan(cfg Config, sel Selection, look PathLookup) (Plan, error) {
	if len(sel.Only) > 0 && len(sel.Except) > 0 {
		return Plan{}, ErrSelectionConflict
	}

	active := Detect(cfg, look)
	active = applyOrder(active, cfg.Order)

	keep := func(name string) bool {
		if sel.Names != nil {
			// Non-nil Names is a verbatim interactive result — including an
			// empty slice, which means "nothing selected".
			return contains(sel.Names, name)
		}
		if len(sel.Only) > 0 {
			return contains(sel.Only, name)
		}
		if len(sel.Except) > 0 {
			return !contains(sel.Except, name)
		}
		return true
	}

	var out []Manager
	for _, m := range active {
		if keep(m.Name) {
			out = append(out, m)
		}
	}
	return Plan{Managers: out}, nil
}

func applyOrder(ms []Manager, order []string) []Manager {
	if len(order) == 0 {
		return ms
	}
	byName := map[string]Manager{}
	for _, m := range ms {
		byName[m.Name] = m
	}
	var out []Manager
	seen := map[string]bool{}
	for _, name := range order {
		if m, ok := byName[name]; ok {
			out = append(out, m)
			seen[name] = true
		}
	}
	for _, m := range ms { // unlisted keep registry order, after ordered
		if !seen[m.Name] {
			out = append(out, m)
		}
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
