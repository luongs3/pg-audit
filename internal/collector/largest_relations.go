package collector

import (
	"context"
	"fmt"
)

// largestRelations reports the biggest relations by total on-disk size
// (table + indexes + TOAST). This is informational: it answers "where is my
// disk going?" and gives context for the bloat and unused-index findings.
func largestRelations(ctx context.Context, pool poolIface) ([]Finding, error) {
	rows, err := pool.Query(ctx, `
		SELECT
		  n.nspname AS schemaname,
		  c.relname AS relname,
		  pg_total_relation_size(c.oid) AS total_bytes,
		  pg_relation_size(c.oid) AS heap_bytes,
		  pg_indexes_size(c.oid) AS index_bytes
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'm', 'p')
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY pg_total_relation_size(c.oid) DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []Finding
	for rows.Next() {
		var schema, table string
		var total, heap, index int64
		if err := rows.Scan(&schema, &table, &total, &heap, &index); err != nil {
			return findings, nil
		}
		findings = append(findings, Finding{
			Severity: Info,
			Title:    fmt.Sprintf("`%s.%s` — %s total", schema, table, humanBytes(total)),
			Detail: fmt.Sprintf(
				"heap %s, indexes %s. Largest relations are where bloat and cache misses cost the most — cross-reference with the bloat and unused-index sections.",
				humanBytes(heap), humanBytes(index),
			),
			Evidence: map[string]any{
				"total_bytes": total,
				"heap_bytes":  heap,
				"index_bytes": index,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return findings, nil
	}
	if len(findings) == 0 {
		return []Finding{{Severity: Info, Title: "No user relations found", Detail: "The database has no user tables outside system schemas."}}, nil
	}
	return findings, nil
}

// humanBytes formats a byte count as a short human-readable string (e.g. 1.4 GB).
func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
