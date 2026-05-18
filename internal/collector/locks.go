package collector

import (
	"context"
	"fmt"
)

// locks captures a point-in-time snapshot of blocking sessions. Useful
// when run during an incident — less useful at quiet times.
func locks(ctx context.Context, pool poolIface) ([]Finding, error) {
	rows, err := pool.Query(ctx, `
		SELECT
		  blocked.pid AS blocked_pid,
		  blocked.usename AS blocked_user,
		  blocking.pid AS blocking_pid,
		  blocking.usename AS blocking_user,
		  blocked.state AS blocked_state,
		  blocked.wait_event_type,
		  blocked.wait_event,
		  COALESCE(SUBSTRING(blocked.query, 1, 120), '') AS blocked_query,
		  COALESCE(SUBSTRING(blocking.query, 1, 120), '') AS blocking_query,
		  EXTRACT(EPOCH FROM (now() - blocked.xact_start))::int AS blocked_xact_age_s
		FROM pg_stat_activity blocked
		JOIN pg_stat_activity blocking
		  ON blocking.pid = ANY(pg_blocking_pids(blocked.pid))
		WHERE blocked.wait_event_type = 'Lock'
		LIMIT 20
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var findings []Finding
	for rows.Next() {
		var blockedPID, blockingPID int32
		var blockedUser, blockingUser, state, waitType, waitEvent, bq, bqb string
		var xactAge int
		if err := rows.Scan(&blockedPID, &blockedUser, &blockingPID, &blockingUser,
			&state, &waitType, &waitEvent, &bq, &bqb, &xactAge); err != nil {
			return nil, err
		}
		findings = append(findings, Finding{
			Severity: Warning,
			Title:    fmt.Sprintf("pid %d (%s) blocked by pid %d (%s), %ds in xact", blockedPID, blockedUser, blockingPID, blockingUser, xactAge),
			Detail:   fmt.Sprintf("Wait: %s/%s\nBlocked query: `%s`\nBlocker query: `%s`", waitType, waitEvent, bq, bqb),
			Evidence: map[string]any{"blocked_pid": blockedPID, "blocking_pid": blockingPID, "xact_age_s": xactAge},
		})
	}

	// Also check for long-idle-in-transaction sessions — silent killer.
	rows2, err := pool.Query(ctx, `
		SELECT pid, usename, state,
		  EXTRACT(EPOCH FROM (now() - xact_start))::int AS xact_age_s,
		  COALESCE(SUBSTRING(query, 1, 120), '')
		FROM pg_stat_activity
		WHERE state = 'idle in transaction'
		  AND xact_start IS NOT NULL
		  AND now() - xact_start > interval '60 seconds'
		ORDER BY xact_start ASC
		LIMIT 10
	`)
	if err != nil {
		return findings, nil
	}
	defer rows2.Close()
	for rows2.Next() {
		var pid int32
		var user, state, q string
		var ageS int
		if err := rows2.Scan(&pid, &user, &state, &ageS, &q); err != nil {
			return findings, nil
		}
		sev := Warning
		if ageS > 600 {
			sev = Critical
		}
		findings = append(findings, Finding{
			Severity: sev,
			Title:    fmt.Sprintf("pid %d idle in transaction for %ds (user %s)", pid, ageS, user),
			Detail:   fmt.Sprintf("Idle-in-transaction sessions hold locks and prevent VACUUM. Investigate the client; consider `idle_in_transaction_session_timeout`. Last query: `%s`", q),
			Evidence: map[string]any{"pid": pid, "age_s": ageS},
		})
	}

	if len(findings) == 0 {
		return []Finding{{Severity: Info, Title: "No blocking or stuck transactions right now", Detail: "Snapshot only — re-run during peak load or an incident."}}, nil
	}
	return findings, nil
}
