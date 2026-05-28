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
- Largest relations by total on-disk size (table + indexes + TOAST)
- Connection usage vs `max_connections`, broken down by state
- Tables without a primary key (schema-health check)

Read-only. Never writes. Never reads row data — only catalog views and
statistics.

## Output formats

Markdown (default) for humans, JSON for pipelines:

```bash
pg-audit run --dsn "$PGAUDIT_DSN"                 # markdown to stdout
pg-audit run --dsn "$PGAUDIT_DSN" --format json   # machine-readable JSON
pg-audit run --dsn "$PGAUDIT_DSN" -f json -o report.json
```

The JSON shape is stable — top-level `database`, `postgres_version`,
`summary` (`critical`/`warning`/`info`/`sections` counts), and a `sections`
array of findings — so you can pipe it into `jq`, store it, or diff it across
runs:

```bash
pg-audit run --dsn "$PGAUDIT_DSN" -f json | jq '.summary'
```

### CI gating

Use `--fail-on` to make `pg-audit` exit non-zero (code 2) when findings
reach a severity threshold — handy as a CI guard against regressions:

```bash
pg-audit run --dsn "$PGAUDIT_DSN" --fail-on critical   # fail the build on any CRITICAL
pg-audit run --dsn "$PGAUDIT_DSN" --fail-on warning    # stricter: warning or above
```

The report still prints normally; only the exit code changes.

## Why this exists

I'm a senior backend engineer. I read these statistics for a living. I got
tired of writing the same SQL queries on every database I inherit, so I
wrote a tool that runs them all at once and dumps a markdown report.

It's MIT-licensed because reading `pg_stat_statements` is not a moat. The
moat is knowing what to do with the output.

## When to use this (vs pghero / pgcenter / pg-utils)

The Postgres-health space has good tools for different shapes of the same problem. `pg-audit` picks one shape: a single binary you run on demand, dumping a markdown report you can paste into a ticket, Slack, or commit. Pick the one that matches your workflow:

- **[pghero](https://github.com/ankane/pghero)** (8.8k★, Ruby/Rails) — long-lived web dashboard you deploy and keep running. Great if you want a UI everyone on the team can bookmark. `pg-audit` is the one-shot CLI alternative when you don’t want a Rails app standing up next to your database.
- **[pgcenter](https://github.com/lesovsky/pgcenter)** (1.6k★, Go) — interactive `top`-style TUI for watching activity live. Great when you’re staring at it during an incident. `pg-audit` is for the async ticket/Slack-paste workflow where the artifact outlives the session.
- **[pg-utils](https://github.com/dataegret/pg-utils)** (1.2k★, shell + SQL) — the same audit playbook as a folder of scripts you run by hand. Great if you want raw control over each query. `pg-audit` packages roughly that playbook behind one command and emits structured (markdown + JSON) output.
- **[pganalyze](https://pganalyze.com) / [pgMustard](https://www.pgmustard.com) / [postgres.ai](https://postgres.ai)** — paid hosted services with continuous monitoring and deep plan analysis. `pg-audit` is the OSS local-CLI alternative for the audit-style workflow (one DB, one report, no agent), not the continuous-monitoring one.

If you already use one of the above and it fits, keep using it. `pg-audit` exists for the "I just inherited this database, what’s broken?" workflow where the deliverable is a markdown file a human reads once and closes.

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

## Example output

Run against a 100k-row demo database with intentionally seeded issues:

```markdown
# Postgres audit: `pgaudit_demo`

_Generated 2026-05-28T11:00:21Z by pg-audit. Postgres version: 16.2._

## Summary
- 0 critical finding(s)
- 7 warning(s)
- 11 section(s) scanned

## Slow queries (pg_stat_statements)

### [INFO] #1: 19 ms mean, 1200 calls, 22281 ms total
```sql
SELECT count(*) FROM orders WHERE amount_cents > $1
```

### [WARNING] #3: 241 ms mean, 1 calls, 241 ms total
```sql
ANALYZE
```

## Bloated tables
### [WARNING] `public.audit_log` — 50.8% bloat (~6776 kB wasted on 13 MB table)
Run `VACUUM (VERBOSE, ANALYZE) public.audit_log;` and review autovacuum settings.

## Missing-index candidates (seq vs idx scans)
### [WARNING] `public.orders` — 3765 seq scans (∞ vs idx), 102 MB table

## Cache hit ratio
### [WARNING] Database cache hit ratio: 94.68%
hits=16102198, disk reads=905296. OLTP workloads usually want >99%.

## Tables without a primary key
### [WARNING] `public.sessions` has no primary key
```

Full example: [demo-report.md](launch/demo-report.md) (also available as [demo-report.json](launch/demo-report.json) for the same run).

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

## Real-world example (1.45 GB OLTP database)

Dogfooded against a production-scale **1.45 GB** OLTP database (table names
redacted). One run, one markdown file:

```markdown
# Postgres audit: `app`
_Postgres version: 16.x. 1 critical finding, 4 warnings, 8 sections scanned._

## Cache hit ratio
### [CRITICAL] Low cache hit ratio (80.49%)
5.76M heap hits / 1.40M disk reads. OLTP wants >99%. shared_buffers is
likely undersized or the cache is cold after restart.

## Bloated tables
### [WARNING] Table is 40.8% bloat
34 MB wasted on an 82 MB table. Candidate for VACUUM (VERBOSE, ANALYZE)
or pg_repack.

## Unused indexes
### [WARNING] 8 unused indexes, ~20 MB reclaimable
0 scans since stats reset, each > 1 MB. Safe DROP INDEX CONCURRENTLY
statements included for each. Every unused index is still maintained on
every write — dropping them cuts write amplification.

## Missing-index candidates
### [WARNING] Hot table doing 985K disk reads at 41.9% cache hit
High seq-read path on a small lookup table — a single correct index (or
fitting it into shared_buffers) turns ~1M disk reads into RAM hits.
```

Across four real databases in that run it surfaced 11 unused indexes, one
40%-bloated table, and two cache-starved hot paths — and caught **two SQL
bugs in the tool itself** (an ambiguous `indexrelid` join and a Postgres-14+
`round(double precision, int)` cast), both fixed before tagging.

## Testing

The hard part of a tool like this isn't the SQL — it's the *judgement* baked
into the thresholds: at what mean time does a query become a warning, at what
bloat ratio is a table worth a human's attention. Get a `>` vs `>=` wrong on
one of those boundaries and the report silently lies.

So every severity decision is pulled out of the SQL-and-row-scanning code
into a pure function in [`internal/collector/classify.go`](internal/collector/classify.go),
and each one is covered by a table-driven test that brackets every threshold
with a just-below / exactly-on / just-above triple. That's where the bugs
live, so that's where the tests are pointed — the classifiers sit at 100%
coverage. The detectors call these functions, so the tests exercise the real
report path, not a parallel copy.

```bash
go test ./... -race -cover
```

CI runs build + vet + race tests on every push.

## Status

v0.1.x — read-only, eight checks implemented, dogfooded on production-scale
data. Issues and PRs welcome.

## License

MIT
