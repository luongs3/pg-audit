package collector

import (
	"context"
	"fmt"
)

// cacheHit reports buffer cache hit ratio per database and per
// user-table. Below 99% on OLTP workloads usually means undersized
// shared_buffers or a cold cache after restart.
func cacheHit(ctx context.Context, pool poolIface) ([]Finding, error) {
	var dbName string
	var hit, read int64
	err := pool.QueryRow(ctx, `
		SELECT datname,
		  COALESCE(blks_hit, 0),
		  COALESCE(blks_read, 0)
		FROM pg_stat_database
		WHERE datname = current_database()
	`).Scan(&dbName, &hit, &read)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	total := hit + read
	if total == 0 {
		return []Finding{{Severity: Info, Title: "No cache stats yet", Detail: "Database has not served any reads since last stats reset."}}, nil
	}
	ratio := float64(hit) / float64(total) * 100
	sev := Info
	switch {
	case ratio < 90:
		sev = Critical
	case ratio < 99:
		sev = Warning
	}
	findings = append(findings, Finding{
		Severity: sev,
		Title:    fmt.Sprintf("Database cache hit ratio: %.2f%%", ratio),
		Detail:   fmt.Sprintf("hits=%d, disk reads=%d. OLTP workloads usually want >99%%. If you just restarted, give it time to warm up. Persistent low ratio → increase shared_buffers or add RAM.", hit, read),
		Evidence: map[string]any{"hit": hit, "read": read, "ratio_pct": ratio},
	})

	// Top tables with the most reads from disk
	rows, err := pool.Query(ctx, `
		SELECT
		  schemaname, relname,
		  heap_blks_hit, heap_blks_read,
		  CASE WHEN (heap_blks_hit + heap_blks_read) = 0
		       THEN 0
		       ELSE round(100.0 * heap_blks_hit / (heap_blks_hit + heap_blks_read), 2)
		  END AS hit_pct
		FROM pg_statio_user_tables
		WHERE heap_blks_read > 100000
		ORDER BY heap_blks_read DESC
		LIMIT 5
	`)
	if err != nil {
		return findings, nil
	}
	defer rows.Close()
	for rows.Next() {
		var schema, table string
		var h, r int64
		var pct float64
		if err := rows.Scan(&schema, &table, &h, &r, &pct); err != nil {
			return findings, nil
		}
		sev := Info
		if pct < 95 {
			sev = Warning
		}
		findings = append(findings, Finding{
			Severity: sev,
			Title:    fmt.Sprintf("`%s.%s` — %.1f%% hit (%d disk reads)", schema, table, pct, r),
			Detail:   "Hot table reaching for disk. Either it's the largest table in the system (expected) or it lacks an index pushing the planner into full reads.",
			Evidence: map[string]any{"hit_pct": pct, "disk_reads": r},
		})
	}

	return findings, nil
}
