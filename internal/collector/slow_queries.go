package collector

import (
	"context"
	"fmt"
)

// slowQueries reads pg_stat_statements (if installed) and surfaces the
// top queries by total execution time. Returns a "skipped" error if the
// extension isn't available — we don't try to install it.
func slowQueries(ctx context.Context, pool poolIface) ([]Finding, error) {
	var hasExt bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements')`,
	).Scan(&hasExt); err != nil {
		return nil, err
	}
	if !hasExt {
		return nil, fmt.Errorf("pg_stat_statements extension not installed (CREATE EXTENSION pg_stat_statements; and add to shared_preload_libraries)")
	}

	rows, err := pool.Query(ctx, `
		SELECT
		  COALESCE(SUBSTRING(query, 1, 200), '') AS query,
		  calls,
		  total_exec_time,
		  mean_exec_time,
		  rows
		FROM pg_stat_statements
		WHERE query NOT ILIKE '%pg_stat_statements%'
		  AND query NOT ILIKE 'BEGIN%'
		  AND query NOT ILIKE 'COMMIT%'
		ORDER BY total_exec_time DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []Finding
	rank := 0
	for rows.Next() {
		rank++
		var q string
		var calls int64
		var totalMs, meanMs float64
		var rowsRet int64
		if err := rows.Scan(&q, &calls, &totalMs, &meanMs, &rowsRet); err != nil {
			return nil, err
		}
		sev := classifySlowQuery(meanMs)
		findings = append(findings, Finding{
			Severity: sev,
			Title:    fmt.Sprintf("#%d: %.0f ms mean, %d calls, %.0f ms total", rank, meanMs, calls, totalMs),
			Detail:   fmt.Sprintf("```sql\n%s\n```\nTuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.", q),
			Evidence: map[string]any{"calls": calls, "mean_ms": meanMs, "total_ms": totalMs, "rows": rowsRet},
		})
	}
	if rank == 0 {
		return []Finding{{Severity: Info, Title: "No statements recorded", Detail: "pg_stat_statements is installed but has no data yet (or was recently reset)."}}, nil
	}
	return findings, rows.Err()
}
