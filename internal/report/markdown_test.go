package report

import (
	"strings"
	"testing"

	"github.com/luongs3/pg-audit/internal/collector"
)

func sampleFindings() *collector.Findings {
	return &collector.Findings{
		DatabaseName: "shop",
		PgVersion:    "16.2",
		Sections: []collector.Section{
			{
				Name: "Cache hit ratio",
				Findings: []collector.Finding{
					{Severity: collector.Critical, Title: "Low cache hit ratio", Detail: "80.49% hit ratio; OLTP wants >99%."},
				},
			},
			{
				Name: "Unused indexes",
				Findings: []collector.Finding{
					{Severity: collector.Warning, Title: "idx_orders_legacy unused", Detail: "0 scans, 4.7 MB reclaimable."},
				},
			},
			{
				Name: "Slow queries (pg_stat_statements)",
				Findings: []collector.Finding{
					{Severity: collector.Info, Title: "Top query by total time", Detail: "SELECT ... 12% of total exec time."},
				},
			},
			{Name: "Replication lag", Skipped: "not a primary"},
			{Name: "Lock-wait hotspots"}, // no findings
		},
	}
}

func TestMarkdownCountsSeverities(t *testing.T) {
	out := Markdown(sampleFindings())

	for _, want := range []string{
		"# Postgres audit: `shop`",
		"Postgres version: 16.2",
		"1 critical finding(s)",
		"1 warning(s)",
		"5 section(s) scanned",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered report missing %q\n---\n%s", want, out)
		}
	}
}

func TestMarkdownRendersSkippedAndEmptySections(t *testing.T) {
	out := Markdown(sampleFindings())

	if !strings.Contains(out, "_Skipped: not a primary_") {
		t.Error("skipped section should render its reason")
	}
	if !strings.Contains(out, "_No findings._") {
		t.Error("section with no findings should render the empty marker")
	}
}

func TestMarkdownUppercasesSeverityLabels(t *testing.T) {
	out := Markdown(sampleFindings())

	for _, want := range []string{"[CRITICAL]", "[WARNING]", "[INFO]"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered report missing severity label %q", want)
		}
	}
}

func TestMarkdownEmptyFindingsIsStable(t *testing.T) {
	out := Markdown(&collector.Findings{DatabaseName: "empty", PgVersion: "16.2"})

	if !strings.Contains(out, "0 critical finding(s)") || !strings.Contains(out, "0 section(s) scanned") {
		t.Errorf("empty findings should still render a summary\n---\n%s", out)
	}
}
