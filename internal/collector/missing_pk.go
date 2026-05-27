package collector

import (
	"context"
	"fmt"
)

// missingPrimaryKey reports ordinary tables that have no primary key.
// Tables without a primary key are a recurring source of operational pain:
// logical replication can't replicate them by default, ORMs and migration
// tools struggle to address individual rows, and accidental duplicate rows
// become hard to dedupe. This is a schema-health signal, not a performance one.
func missingPrimaryKey(ctx context.Context, pool poolIface) ([]Finding, error) {
	rows, err := pool.Query(ctx, `
		SELECT n.nspname AS schemaname, c.relname AS relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r'
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND NOT EXISTS (
		    SELECT 1 FROM pg_constraint con
		    WHERE con.conrelid = c.oid AND con.contype = 'p'
		  )
		ORDER BY n.nspname, c.relname
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []Finding
	for rows.Next() {
		var schema, table string
		if err := rows.Scan(&schema, &table); err != nil {
			return findings, nil
		}
		findings = append(findings, Finding{
			Severity: classifyMissingPrimaryKey(),
			Title:    fmt.Sprintf("`%s.%s` has no primary key", schema, table),
			Detail:   "Tables without a primary key can't be replicated by logical replication (without REPLICA IDENTITY FULL), are awkward for ORMs and row-level updates, and make duplicate rows hard to detect. Add a primary key or a UNIQUE NOT NULL identity column.",
			Evidence: map[string]any{"schema": schema, "table": table},
		})
	}
	if err := rows.Err(); err != nil {
		return findings, nil
	}
	return findings, nil
}
