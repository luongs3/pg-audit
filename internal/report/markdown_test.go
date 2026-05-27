package report

import (
	"strings"
	"testing"

	"github.com/luongs3/pg-audit/internal/collector"
)

// sampleFindings builds a Findings tree exercising the cases the renderer
// has to handle: a section with mixed severities, a skipped section, and an
// empty section.
func sampleFindings() *collector.Findings {
	return &collector.Findings{
		DatabaseName: "shop",
		PgVersion:    "16.2",
		Sections: []collector.Section{
			{
				Name: "Slow queries",
				Findings: []collector.Finding{
					{Severity: collector.Critical, Title: "q1 slow", Detail: "..."},
					{Severity: collector.Warning, Title: "q2 warm", Detail: "..."},
					{Severity: collector.Info, Title: "q3 fine", Detail: "..."},
				},
			},
			{
				Name:    "Slow queries (pg_stat_statements)",
				Skipped: "extension not installed",
			},
			{
				Name:     "Replication lag",
				Findings: nil, // empty → "_No findings._"
			},
		},
	}
}

func TestMarkdownSummaryCounts(t *testing.T) {
	md := Markdown(sampleFindings())
	// One critical and one warning across all sections.
	if !strings.Contains(md, "1 critical finding(s)") {
		t.Errorf("expected 1 critical in summary, got:\n%s", md)
	}
	if !strings.Contains(md, "1 warning(s)") {
		t.Errorf("expected 1 warning in summary, got:\n%s", md)
	}
	if !strings.Contains(md, "3 section(s) scanned") {
		t.Errorf("expected 3 sections in summary, got:\n%s", md)
	}
}

func TestMarkdownRendersSkippedAndEmpty(t *testing.T) {
	md := Markdown(sampleFindings())
	if !strings.Contains(md, "_Skipped: extension not installed_") {
		t.Errorf("skipped section not rendered:\n%s", md)
	}
	if !strings.Contains(md, "_No findings._") {
		t.Errorf("empty section not rendered:\n%s", md)
	}
}

func TestMarkdownIncludesDBNameAndVersion(t *testing.T) {
	md := Markdown(sampleFindings())
	if !strings.Contains(md, "shop") {
		t.Errorf("database name missing from report:\n%s", md)
	}
	if !strings.Contains(md, "16.2") {
		t.Errorf("pg version missing from report:\n%s", md)
	}
}

func TestMarkdownUppercasesSeverityTags(t *testing.T) {
	md := Markdown(sampleFindings())
	if !strings.Contains(md, "[CRITICAL]") {
		t.Errorf("expected uppercased [CRITICAL] tag:\n%s", md)
	}
}
