package collector

import (
	"context"
	"fmt"
)

// bloat uses the well-known pgstattuple-free approximate-bloat query
// (originally from check_postgres / ioguix). It estimates wasted space
// in tables based on column statistics — fast, no extension required.
func bloat(ctx context.Context, pool poolIface) ([]Finding, error) {
	q := `
WITH constants AS (
  SELECT current_setting('block_size')::numeric AS bs, 23 AS hdr, 8 AS ma
), no_stats AS (
  SELECT table_schema, table_name,
    n_live_tup::numeric AS est_rows,
    pg_table_size(relid)::numeric AS table_size
  FROM information_schema.columns
  JOIN pg_stat_user_tables AS psut
    ON table_schema = psut.schemaname AND table_name = psut.relname
  LEFT OUTER JOIN pg_stats
    ON table_schema = pg_stats.schemaname
    AND table_name = pg_stats.tablename
    AND column_name = attname
  WHERE attname IS NULL AND table_schema NOT IN ('pg_catalog', 'information_schema')
  GROUP BY table_schema, table_name, relid, n_live_tup
), null_headers AS (
  SELECT hdr + 1 + (sum(case when null_frac <> 0 THEN 1 else 0 END)/8) AS nullhdr,
    SUM((1-null_frac)*avg_width) AS datawidth,
    MAX(null_frac) AS maxfracsum,
    schemaname, tablename, hdr, ma, bs
  FROM pg_stats CROSS JOIN constants
  LEFT OUTER JOIN no_stats
    ON schemaname = no_stats.table_schema AND tablename = no_stats.table_name
  WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    AND no_stats.table_name IS NULL
    AND EXISTS (SELECT 1 FROM information_schema.columns
        WHERE schemaname = columns.table_schema AND tablename = columns.table_name)
  GROUP BY schemaname, tablename, hdr, ma, bs
), data_headers AS (
  SELECT ma, bs, hdr, schemaname, tablename,
    (datawidth+(hdr+ma-(case when hdr%ma=0 THEN ma ELSE hdr%ma END)))::numeric AS datahdr,
    (maxfracsum*(nullhdr+ma-(case when nullhdr%ma=0 THEN ma ELSE nullhdr%ma END))) AS nullhdr2
  FROM null_headers
), table_estimates AS (
  SELECT schemaname, tablename, bs,
    reltuples::numeric AS est_rows, relpages * bs AS table_bytes,
    CEIL((reltuples*(datahdr+nullhdr2+4+ma-(CASE WHEN datahdr%ma=0 THEN ma ELSE datahdr%ma END))/(bs-20))) * bs AS expected_bytes,
    reltoastrelid
  FROM data_headers
  JOIN pg_class ON tablename = relname
  JOIN pg_namespace ON relnamespace = pg_namespace.oid AND schemaname = nspname
  WHERE pg_class.relkind = 'r'
)
SELECT schemaname, tablename,
  pg_size_pretty(table_bytes) AS table_size,
  pg_size_pretty((table_bytes - expected_bytes)::bigint) AS bloat_size,
  CASE WHEN table_bytes > 0
    THEN round((100 * (table_bytes - expected_bytes) / table_bytes)::numeric, 1)
    ELSE 0 END AS bloat_pct,
  (table_bytes - expected_bytes)::bigint AS bloat_bytes
FROM table_estimates
WHERE table_bytes > 10*1024*1024  -- > 10 MB
  AND (table_bytes - expected_bytes) > 0
ORDER BY (table_bytes - expected_bytes) DESC
LIMIT 20
`
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var findings []Finding
	for rows.Next() {
		var schema, table, sizeP, bloatSizeP string
		var bloatPct float64
		var bloatBytes int64
		if err := rows.Scan(&schema, &table, &sizeP, &bloatSizeP, &bloatPct, &bloatBytes); err != nil {
			return nil, err
		}
		sev := Info
		if bloatPct >= 50 && bloatBytes > 100*1024*1024 {
			sev = Critical
		} else if bloatPct >= 25 {
			sev = Warning
		} else {
			continue
		}
		findings = append(findings, Finding{
			Severity: sev,
			Title:    fmt.Sprintf("`%s.%s` — %.1f%% bloat (~%s wasted on %s table)", schema, table, bloatPct, bloatSizeP, sizeP),
			Detail:   fmt.Sprintf("Run `VACUUM (VERBOSE, ANALYZE) %s.%s;` and review autovacuum settings (`autovacuum_vacuum_scale_factor`). If bloat persists, schedule `pg_repack` during a low-traffic window — it rewrites the table without an exclusive lock.", schema, table),
			Evidence: map[string]any{"bloat_pct": bloatPct, "bloat_bytes": bloatBytes},
		})
	}
	if len(findings) == 0 {
		return []Finding{{Severity: Info, Title: "No significant bloat (>25%) on tables >10 MB", Detail: "Estimate based on pg_stats column widths. For precise numbers install pgstattuple."}}, nil
	}
	return findings, rows.Err()
}
