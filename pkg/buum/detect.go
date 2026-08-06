package buum

import "os/exec"

// DefaultLookup reports whether bin is resolvable on the current PATH.
var DefaultLookup PathLookup = func(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// Detect returns enabled managers whose DetectBin is found by look.
func Detect(cfg Config, look PathLookup) []Manager {
	var out []Manager
	for _, m := range cfg.Managers {
		if !m.Enabled {
			continue
		}
		if look(m.DetectBin) {
			out = append(out, m)
		}
	}
	return out
}
