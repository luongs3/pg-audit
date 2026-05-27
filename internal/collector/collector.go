package collector

import (
	"context"

	"github.com/luongs3/pg-audit/internal/db"
)

type Severity string

const (
	Critical Severity = "critical"
	Warning  Severity = "warning"
	Info     Severity = "info"
)

type Finding struct {
	Section  string
	Severity Severity
	Title    string
	Detail   string
	Evidence map[string]any
}

type Findings struct {
	DatabaseName string
	PgVersion    string
	Sections     []Section
}

type Section struct {
	Name     string
	Findings []Finding
	Skipped  string
}

// RunAll connects to the DB and runs every check. Returns a Findings tree
// that report.Markdown renders to a markdown document.
func RunAll(ctx context.Context, dsn string) (*Findings, error) {
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer pool.Close()

	f := &Findings{}
	if err := collectMeta(ctx, pool, f); err != nil {
		return nil, err
	}

	// Each check is independent; failure in one shouldn't abort the rest.
	for _, c := range allChecks() {
		section := Section{Name: c.name}
		findings, err := c.run(ctx, pool)
		if err != nil {
			section.Skipped = err.Error()
		} else {
			section.Findings = findings
		}
		f.Sections = append(f.Sections, section)
	}
	return f, nil
}

type check struct {
	name string
	run  func(ctx context.Context, pool poolIface) ([]Finding, error)
}

// allChecks is the ordered list of audit sections. Each is implemented in its
// own file in this package (slow_queries.go, unused_indexes.go, ...).
func allChecks() []check {
	return []check{
		{name: "Slow queries (pg_stat_statements)", run: slowQueries},
		{name: "Unused indexes", run: unusedIndexes},
		{name: "Bloated tables", run: bloat},
		{name: "Missing-index candidates (seq vs idx scans)", run: missingIndexes},
		{name: "Lock-wait hotspots", run: locks},
		{name: "Config smells", run: configSmells},
		{name: "Cache hit ratio", run: cacheHit},
		{name: "Replication lag", run: replicationLag},
	}
}
