package collector

import (
	"context"
	"fmt"
)

// unusedIndexes finds indexes that have never been scanned and are larger
// than a minimum size. Dropping unused indexes reclaims disk and speeds up
// writes (every index has to be maintained on INSERT/UPDATE/DELETE).
func unusedIndexes(ctx context.Context, pool poolIface) ([]Finding, error) {
	rows, err := pool.Query(ctx, `
		SELECT
		  schemaname,
		  relname AS table_name,
		  indexrelname AS index_name,
		  idx_scan,
		  pg_size_pretty(pg_relation_size(ui.indexrelid)) AS size_pretty,
		  pg_relation_size(ui.indexrelid) AS size_bytes
		FROM pg_stat_user_indexes ui
		JOIN pg_index i ON i.indexrelid = ui.indexrelid
		WHERE idx_scan = 0
		  AND NOT i.indisunique
		  AND NOT i.indisprimary
		  AND pg_relation_size(ui.indexrelid) > 1024 * 1024  -- > 1 MB
		ORDER BY pg_relation_size(ui.indexrelid) DESC
		LIMIT 30
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []Finding
	var totalBytes int64
	for rows.Next() {
		var schema, table, idx, sizePretty string
		var scans, sizeBytes int64
		if err := rows.Scan(&schema, &table, &idx, &scans, &sizePretty, &sizeBytes); err != nil {
			return nil, err
		}
		totalBytes += sizeBytes
		sev := Info
		if sizeBytes > 100*1024*1024 {
			sev = Warning
		}
		findings = append(findings, Finding{
			Severity: sev,
			Title:    fmt.Sprintf("`%s.%s.%s` — %s, 0 scans", schema, table, idx, sizePretty),
			Detail:   fmt.Sprintf("Index has never been used since stats were last reset. Verify uptime with `SELECT stats_reset FROM pg_stat_database WHERE datname = current_database()` before dropping. Safe drop: `DROP INDEX CONCURRENTLY %s.%s;`", schema, idx),
			Evidence: map[string]any{"size_bytes": sizeBytes, "scans": scans},
		})
	}
	if len(findings) == 0 {
		return []Finding{{Severity: Info, Title: "No unused indexes >1 MB", Detail: "Every non-unique, non-primary index has been scanned at least once."}}, nil
	}
	// Bump severity of the headline if total reclaimable space is large.
	if totalBytes > 1024*1024*1024 {
		findings = append([]Finding{{
			Severity: Critical,
			Title:    fmt.Sprintf("~%d MB total reclaimable", totalBytes/1024/1024),
			Detail:   "Total disk reclaimable by dropping the unused indexes below. Bigger upside: faster writes, faster autovacuum.",
		}}, findings...)
	}
	return findings, rows.Err()
}
