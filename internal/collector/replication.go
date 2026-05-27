package collector

import (
	"context"
	"fmt"
)

// replicationLag reports lag against any connected replicas. Returns a
// single "no replicas" info finding when none are connected.
func replicationLag(ctx context.Context, pool poolIface) ([]Finding, error) {
	rows, err := pool.Query(ctx, `
		SELECT
		  COALESCE(client_addr::text, 'local'),
		  COALESCE(application_name, ''),
		  state,
		  sync_state,
		  COALESCE(EXTRACT(EPOCH FROM write_lag)::numeric, 0),
		  COALESCE(EXTRACT(EPOCH FROM flush_lag)::numeric, 0),
		  COALESCE(EXTRACT(EPOCH FROM replay_lag)::numeric, 0),
		  COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn), 0)::bigint
		FROM pg_stat_replication
	`)
	if err != nil {
		// pg_current_wal_lsn() fails on replicas — try the read-only variant.
		return []Finding{{Severity: Info, Title: "Replication check skipped", Detail: fmt.Sprintf("Possibly running on a replica or pre-PG10: %s", err.Error())}}, nil
	}
	defer rows.Close()
	var findings []Finding
	found := false
	for rows.Next() {
		found = true
		var addr, appName, state, sync string
		var writeLag, flushLag, replayLag float64
		var lagBytes int64
		if err := rows.Scan(&addr, &appName, &state, &sync, &writeLag, &flushLag, &replayLag, &lagBytes); err != nil {
			return findings, nil
		}
		sev := classifyReplicationLag(replayLag, lagBytes)
		findings = append(findings, Finding{
			Severity: sev,
			Title:    fmt.Sprintf("Replica `%s` (%s) — %.1fs replay lag, %d KB behind", addr, appName, replayLag, lagBytes/1024),
			Detail:   fmt.Sprintf("state=%s, sync_state=%s, write_lag=%.1fs, flush_lag=%.1fs", state, sync, writeLag, flushLag),
			Evidence: map[string]any{"replay_lag_s": replayLag, "lag_bytes": lagBytes},
		})
	}
	if !found {
		return []Finding{{Severity: Info, Title: "No replicas connected", Detail: "Either there are no replicas, or none were connected at the moment of the audit. Not necessarily a problem."}}, nil
	}
	return findings, nil
}
