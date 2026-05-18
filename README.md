# pg-audit

A read-only Postgres health check. One binary, one connection string, one
markdown report.

```bash
pg-audit run --dsn "postgres://user:pass@host:5432/db?sslmode=require"
```

You get a markdown file covering:

- Top 20 slow queries from `pg_stat_statements` (by total time + by mean time)
- Unused indexes (zero scans, large size)
- Bloated tables (estimated dead tuple ratio)
- Lock-wait hotspots
- Config smells (`shared_buffers`, `work_mem`, `effective_cache_size`,
  `random_page_cost`, `max_connections` vs `shared_buffers` ratio,
  autovacuum off, log_min_duration_statement unset)
- Missing-index candidates (high `seq_scan` to `idx_scan` ratio on large tables)
- Replication lag (if applicable)
- Cache hit ratio per database

Read-only. Never writes. Never reads row data — only catalog views and
statistics.

## Why this exists

I'm a senior backend engineer. I read these statistics for a living. I got
tired of writing the same SQL queries on every database I inherit, so I
wrote a tool that runs them all at once and dumps a markdown report.

It's MIT-licensed because reading `pg_stat_statements` is not a moat. The
moat is knowing what to do with the output.

## Want a human to read your output?

`pg-audit` tells you what's wrong. It does not tell you what to fix
**first**, why some recommendations are dangerous on your specific
workload, or how to roll the fix out without downtime.

If you want a senior backend engineer to read your `pg-audit` output and
send back a prioritized fix plan with rollout steps:

- $800 flat, 48-hour turnaround
- PDF report, no subscription, no enterprise sales call
- Read-only credential, time-boxed
- Refund if I don't find at least 3 actionable wins

Email **luongr3@gmail.com** with the output of `pg-audit run` and a one-line description of your workload.

## Install

```bash
go install github.com/luongs3/pg-audit/cmd/pg-audit@latest
```

Or download a release binary from the Releases page.

## Required Postgres extensions

`pg_stat_statements` must be enabled. If it's not, `pg-audit` will tell you
how to enable it and skip that section.

## Permissions

Connect as a user with `pg_monitor` role (recommended) or any user with
`SELECT` on the system catalog views. `pg-audit` issues `SELECT` only.

## What it doesn't do

- Live monitoring (use [pganalyze](https://pganalyze.com) if you want a SaaS)
- Query rewriting (use [EverSQL](https://eversql.com) inside Aiven)
- Index recommendations beyond "this index is unused" (the actually-hard
  recommendation is the human-judgement part — that's the paid layer)

## Status

v0.x, dogfooded on a production-scale 1.45 GB database that surfaced
8 unused indexes, a 40% bloated table, and an 80% cache hit ratio.
Two SQL bugs in the tool itself were caught and fixed by that same
dogfood run. Issues and PRs welcome.

## License

MIT
