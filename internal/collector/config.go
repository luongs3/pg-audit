package collector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// configSmells checks a handful of Postgres settings against
// pragmatic defaults. Not exhaustive — flags only the things that
// commonly bite teams who never tuned past the install defaults.
func configSmells(ctx context.Context, pool poolIface) ([]Finding, error) {
	settings, err := readSettings(ctx, pool)
	if err != nil {
		return nil, err
	}

	totalRAM, _ := readTotalRAMBytes(ctx, pool)

	var findings []Finding

	// shared_buffers: typical advice is 25% of RAM, but anything < 128 MB is almost certainly default-and-forgot.
	if v, ok := bytesSetting(settings, "shared_buffers"); ok {
		if v < 128*1024*1024 {
			findings = append(findings, Finding{
				Severity: Warning,
				Title:    "shared_buffers is very small (" + settings["shared_buffers"] + ")",
				Detail:   "Default is 128 MB. On a dedicated DB server, 25% of RAM is the usual starting point. If this is a managed DB and you can't change it, ignore.",
			})
		}
		if totalRAM > 0 && float64(v)/float64(totalRAM) < 0.10 {
			findings = append(findings, Finding{
				Severity: Info,
				Title:    fmt.Sprintf("shared_buffers (%s) is < 10%% of detected RAM (%d MB)", settings["shared_buffers"], totalRAM/1024/1024),
				Detail:   "Consider raising toward 25% of RAM on a dedicated DB host.",
			})
		}
	}

	// work_mem: tiny default (4 MB) is a common cause of sorts spilling to disk.
	if v, ok := bytesSetting(settings, "work_mem"); ok && v <= 4*1024*1024 {
		findings = append(findings, Finding{
			Severity: Info,
			Title:    "work_mem at default (4 MB)",
			Detail:   "Many analytical queries spill to disk at this level. Bump per-session (`SET LOCAL work_mem = '64MB';`) for heavy reports, or globally if RAM allows. Watch out: applied per sort/hash node, not per query.",
		})
	}

	// max_connections: > 200 with a small pool is a smell — connection pooling (PgBouncer) usually wins.
	if v, ok := intSetting(settings, "max_connections"); ok && v > 200 {
		findings = append(findings, Finding{
			Severity: Warning,
			Title:    fmt.Sprintf("max_connections = %d", v),
			Detail:   "Each connection costs ~10 MB and contends on locks. Above ~200, you almost always want PgBouncer (transaction pooling) and a smaller max_connections (50-100).",
		})
	}

	// effective_cache_size: should be ~50-75% of RAM. Default 4 GB on PG 12+; on small boxes it's wrong the other way.
	if v, ok := bytesSetting(settings, "effective_cache_size"); ok {
		if totalRAM > 0 && float64(v)/float64(totalRAM) < 0.30 {
			findings = append(findings, Finding{
				Severity: Info,
				Title:    fmt.Sprintf("effective_cache_size (%s) is < 30%% of RAM", settings["effective_cache_size"]),
				Detail:   "This is a planner hint, not a buffer. Setting it too low pushes the planner toward seq scans. Recommend 50-75% of total RAM.",
			})
		}
	}

	// autovacuum is on by default — flag only if explicitly turned off.
	if v, ok := settings["autovacuum"]; ok && strings.EqualFold(v, "off") {
		findings = append(findings, Finding{
			Severity: Critical,
			Title:    "autovacuum is DISABLED",
			Detail:   "Without autovacuum, tables bloat indefinitely and xid wraparound becomes a real risk. Turn it back on and tune `autovacuum_vacuum_scale_factor` if it's noisy.",
		})
	}

	// log_min_duration_statement: -1 means slow-query logging is off.
	if v, ok := settings["log_min_duration_statement"]; ok && v == "-1" {
		findings = append(findings, Finding{
			Severity: Info,
			Title:    "Slow-query logging is off",
			Detail:   "Set `log_min_duration_statement = 1000` (ms) to capture queries over 1s in the server log. Free observability.",
		})
	}

	if len(findings) == 0 {
		return []Finding{{Severity: Info, Title: "No obvious config smells", Detail: "Spot-check only — doesn't mean the config is optimal."}}, nil
	}
	return findings, nil
}

func readSettings(ctx context.Context, pool poolIface) (map[string]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT name, setting, unit
		FROM pg_settings
		WHERE name IN (
		  'shared_buffers','work_mem','maintenance_work_mem','effective_cache_size',
		  'max_connections','autovacuum','log_min_duration_statement',
		  'wal_level','random_page_cost','default_statistics_target'
		)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var name, setting string
		var unit *string
		if err := rows.Scan(&name, &setting, &unit); err != nil {
			return nil, err
		}
		if unit != nil && *unit != "" {
			m[name] = setting + *unit
		} else {
			m[name] = setting
		}
	}
	return m, rows.Err()
}

func readTotalRAMBytes(ctx context.Context, pool poolIface) (int64, error) {
	// Best-effort. Server-side RAM isn't introspectable from PG — caller
	// gets 0 if unknown.
	return 0, nil
}

func bytesSetting(m map[string]string, key string) (int64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	v = strings.TrimSpace(v)
	mult := int64(1)
	switch {
	case strings.HasSuffix(v, "kB"):
		mult = 1024
		v = strings.TrimSuffix(v, "kB")
	case strings.HasSuffix(v, "MB"):
		mult = 1024 * 1024
		v = strings.TrimSuffix(v, "MB")
	case strings.HasSuffix(v, "GB"):
		mult = 1024 * 1024 * 1024
		v = strings.TrimSuffix(v, "GB")
	case strings.HasSuffix(v, "8kB"):
		mult = 8 * 1024
		v = strings.TrimSuffix(v, "8kB")
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0, false
	}
	return n * mult, true
}

func intSetting(m map[string]string, key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, false
	}
	return n, true
}
