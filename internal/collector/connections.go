package collector

import (
	"context"
	"fmt"
)

// connections reports how many backend connections are in use relative to
// max_connections, and breaks them down by state. Exhausting connection slots
// causes hard "sorry, too many clients already" failures, so this surfaces
// saturation before it becomes an outage — a strong signal that a connection
// pooler (PgBouncer) or a lower per-app pool size is needed.
func connections(ctx context.Context, pool poolIface) ([]Finding, error) {
	var maxConns, used int
	if err := pool.QueryRow(ctx, `SELECT setting::int FROM pg_settings WHERE name = 'max_connections'`).Scan(&maxConns); err != nil {
		return nil, err
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity`).Scan(&used); err != nil {
		return nil, err
	}
	if maxConns == 0 {
		return []Finding{{Severity: Info, Title: "max_connections unavailable", Detail: "Could not read max_connections."}}, nil
	}

	usedPct := float64(used) / float64(maxConns) * 100
	findings := []Finding{{
		Severity: classifyConnectionUsage(usedPct),
		Title:    fmt.Sprintf("%d / %d connections in use (%.0f%%)", used, maxConns, usedPct),
		Detail:   "Approaching max_connections risks hard \"too many clients\" failures. If usage is high with many idle connections, put a pooler (PgBouncer) in front or lower per-app pool sizes rather than raising max_connections (each backend costs memory).",
		Evidence: map[string]any{"used": used, "max_connections": maxConns, "used_pct": usedPct},
	}}

	// Breakdown by state — a pile of "idle" connections is the classic sign of
	// an oversized application pool holding slots it isn't using.
	rows, err := pool.Query(ctx, `
		SELECT COALESCE(state, 'unknown') AS state, count(*)
		FROM pg_stat_activity
		WHERE state IS NOT NULL
		GROUP BY state
		ORDER BY count(*) DESC
	`)
	if err != nil {
		return findings, nil
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		var n int
		if err := rows.Scan(&state, &n); err != nil {
			return findings, nil
		}
		findings = append(findings, Finding{
			Severity: Info,
			Title:    fmt.Sprintf("%d connection(s) in state %q", n, state),
			Evidence: map[string]any{"state": state, "count": n},
		})
	}
	return findings, nil
}
