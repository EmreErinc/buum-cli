package ui

import (
	"encoding/json"
	"io"

	"github.com/emreerinc/buum/pkg/buum"
)

// JSONWriter emits newline-delimited JSON events for --json mode.
type JSONWriter struct {
	W io.Writer
}

func (j *JSONWriter) emit(v any) {
	b, _ := json.Marshal(v)
	j.W.Write(b)
	j.W.Write([]byte("\n"))
}

// Prompt emits a prompt event when a child blocks on input.
func (j *JSONWriter) Prompt(manager string, step []string, text string) {
	j.emit(map[string]any{
		"event":   "prompt",
		"manager": manager,
		"step":    step,
		"text":    text,
	})
}

// ReportEvent emits the final run report.
func (j *JSONWriter) ReportEvent(rep buum.Report) {
	j.emit(struct {
		Event   string               `json:"event"`
		Results []buum.ManagerResult `json:"results"`
		Summary buum.Summary         `json:"summary"`
	}{"report", rep.Results, rep.Summary})
}

type jsonManager struct {
	Name      string `json:"name"`
	DetectBin string `json:"detect_bin"`
	NeedsSudo bool   `json:"needs_sudo"`
}

func toJSONManagers(ms []buum.Manager) []jsonManager {
	out := make([]jsonManager, 0, len(ms))
	for _, m := range ms {
		out = append(out, jsonManager{m.Name, m.DetectBin, m.NeedsSudo})
	}
	return out
}

// List emits the detected/available sets for `buum list --json`.
func (j *JSONWriter) List(detected, available []buum.Manager) {
	j.emit(map[string]any{
		"detected":  toJSONManagers(detected),
		"available": toJSONManagers(available),
	})
}
