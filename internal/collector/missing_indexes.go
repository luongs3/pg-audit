package collector

import (
	"context"
	"fmt"
)

// missingIndexes flags tables where sequential scans dwarf index scans
// AND the table is big enough that a seq scan hurts. Doesn't tell you
// which column to index — only that the planner is reaching for seq scans.
func missingIndexes(ctx context.Context, pool poolIface) ([]Finding, error) {
	rows, err := pool.Query(ctx, `
		SELECT
		  schemaname,
		  relname AS table_name,
		  seq_scan,
		  idx_scan,
		  seq_tup_read,
		  n_live_tup,
		  pg_size_pretty(pg_relation_size(relid)) AS table_size,
		  pg_relation_size(relid) AS size_bytes
		FROM pg_stat_user_tables
		WHERE seq_scan > 1000
		  AND seq_scan > COALESCE(idx_scan, 0) * 2
		  AND pg_relation_size(relid) > 50 * 1024 * 1024  -- > 50 MB
		  AND n_live_tup > 10000
		ORDER BY seq_tup_read DESC
		LIMIT 15
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var findings []Finding
	for rows.Next() {
		var schema, table, sizeP string
		var seqScan, idxScan, seqTupRead, nLive, sizeB int64
		if err := rows.Scan(&schema, &table, &seqScan, &idxScan, &seqTupRead, &nLive, &sizeP, &sizeB); err != nil {
			return nil, err
		}
		sev := classifyMissingIndex(seqScan, sizeB)
		ratio := "∞"
		if idxScan > 0 {
			ratio = fmt.Sprintf("%.1fx", float64(seqScan)/float64(idxScan))
		}
		findings = append(findings, Finding{
			Severity: sev,
			Title:    fmt.Sprintf("`%s.%s` — %d seq scans (%s vs idx), %s table", schema, table, seqScan, ratio, sizeP),
			Detail:   fmt.Sprintf("Identify the hot query with `SELECT query FROM pg_stat_statements WHERE query ILIKE '%%%s%%' ORDER BY total_exec_time DESC LIMIT 5;`, then add an index on the columns in its WHERE/JOIN. Verify with `EXPLAIN ANALYZE` before and after.", table),
			Evidence: map[string]any{"seq_scan": seqScan, "idx_scan": idxScan, "size_bytes": sizeB},
		})
	}
	if len(findings) == 0 {
		return []Finding{{Severity: Info, Title: "No tables with concerning seq-scan ratios", Detail: "Every table >50 MB is being hit by index scans more than sequential scans."}}, nil
	}
	return findings, rows.Err()
}
