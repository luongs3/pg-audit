package report

import (
	"encoding/json"

	"github.com/luongs3/pg-audit/internal/collector"
)

// jsonReport is the stable machine-readable shape of an audit run. It is a
// deliberately flat, tool-friendly projection of collector.Findings so the
// output can be piped into jq, stored, or diffed across runs.
type jsonReport struct {
	Database string        `json:"database"`
	Version  string        `json:"postgres_version"`
	Summary  jsonSummary   `json:"summary"`
	Sections []jsonSection `json:"sections"`
}

type jsonSummary struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
	Sections int `json:"sections"`
}

type jsonSection struct {
	Name     string        `json:"name"`
	Skipped  string        `json:"skipped,omitempty"`
	Findings []jsonFinding `json:"findings"`
}

type jsonFinding struct {
	Severity string         `json:"severity"`
	Title    string         `json:"title"`
	Detail   string         `json:"detail"`
	Evidence map[string]any `json:"evidence,omitempty"`
}

// JSON renders findings as an indented JSON document with a trailing newline.
func JSON(f *collector.Findings) (string, error) {
	r := jsonReport{
		Database: f.DatabaseName,
		Version:  f.PgVersion,
		Summary: jsonSummary{
			Critical: count(f, collector.Critical),
			Warning:  count(f, collector.Warning),
			Info:     count(f, collector.Info),
			Sections: len(f.Sections),
		},
		Sections: make([]jsonSection, 0, len(f.Sections)),
	}
	for _, s := range f.Sections {
		js := jsonSection{
			Name:     s.Name,
			Skipped:  s.Skipped,
			Findings: make([]jsonFinding, 0, len(s.Findings)),
		}
		for _, x := range s.Findings {
			js.Findings = append(js.Findings, jsonFinding{
				Severity: string(x.Severity),
				Title:    x.Title,
				Detail:   x.Detail,
				Evidence: x.Evidence,
			})
		}
		r.Sections = append(r.Sections, js)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
