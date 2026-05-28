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
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

### [INFO] #2: 24 ms mean, 50 calls, 1203 ms total

```sql
SELECT count(*) FROM orders o JOIN users u ON u.id=o.user_id WHERE u.country = $1
```
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

### [WARNING] #3: 241 ms mean, 1 calls, 241 ms total

```sql
ANALYZE
```
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

### [INFO] #4: 1 ms mean, 200 calls, 136 ms total

```sql
SELECT count(*) FROM users WHERE email = $1
```
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

### [INFO] #5: 25 ms mean, 5 calls, 126 ms total

```sql
SELECT count(*) FROM orders o
      WHERE o.amount_cents > $1
        AND o.user_id IN (SELECT id FROM users WHERE email LIKE $2)
```
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

### [INFO] #6: 3 ms mean, 20 calls, 63 ms total

```sql
SELECT count(*) FROM audit_log WHERE action = $1
```
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

### [INFO] #7: 0 ms mean, 200 calls, 1 ms total

```sql
SELECT display_name FROM users WHERE id = $1
```
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

### [INFO] #8: 0 ms mean, 1 calls, 0 ms total

```sql
SELECT schemaname, relname, seq_scan, idx_scan, n_live_tup,
           pg_relation_size(relid), pg_size_pretty(pg_relation_size(relid))
    FROM pg_stat_user_tables ORDER BY seq_scan DESC
```
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

### [INFO] #9: 0 ms mean, 1 calls, 0 ms total

```sql
SELECT schemaname, relname, seq_scan, idx_scan, pg_relation_size(relid)
    FROM pg_stat_user_tables
    WHERE seq_scan > $1
      AND seq_scan > COALESCE(idx_scan, $2) * $3
      AND pg_relation_size
```
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

### [INFO] #10: 0 ms mean, 1 calls, 0 ms total

```sql
SELECT EXISTS (SELECT $1 FROM pg_extension WHERE extname = $2)
```
Tuning hints: check the plan with EXPLAIN (ANALYZE, BUFFERS); look for Seq Scan on large tables, missing indexes on the join/filter columns, or unbounded LIMIT.

## Unused indexes

### [INFO] `public.orders.idx_orders_created_unused` — 13 MB, 0 scans

Index has never been used since stats were last reset. Verify uptime with `SELECT stats_reset FROM pg_stat_database WHERE datname = current_database()` before dropping. Safe drop: `DROP INDEX CONCURRENTLY public.idx_orders_created_unused;`

### [INFO] `public.orders.idx_orders_notes_unused` — 4720 kB, 0 scans

Index has never been used since stats were last reset. Verify uptime with `SELECT stats_reset FROM pg_stat_database WHERE datname = current_database()` before dropping. Safe drop: `DROP INDEX CONCURRENTLY public.idx_orders_notes_unused;`

### [INFO] `public.orders.idx_orders_status_unused` — 4112 kB, 0 scans

Index has never been used since stats were last reset. Verify uptime with `SELECT stats_reset FROM pg_stat_database WHERE datname = current_database()` before dropping. Safe drop: `DROP INDEX CONCURRENTLY public.idx_orders_status_unused;`

## Bloated tables

### [WARNING] `public.audit_log` — 50.8% bloat (~6776 kB wasted on 13 MB table)

Run `VACUUM (VERBOSE, ANALYZE) public.audit_log;` and review autovacuum settings (`autovacuum_vacuum_scale_factor`). If bloat persists, schedule `pg_repack` during a low-traffic window — it rewrites the table without an exclusive lock.

## Missing-index candidates (seq vs idx scans)

### [WARNING] `public.orders` — 3765 seq scans (∞ vs idx), 102 MB table

Identify the hot query with `SELECT query FROM pg_stat_statements WHERE query ILIKE '%orders%' ORDER BY total_exec_time DESC LIMIT 5;`, then add an index on the columns in its WHERE/JOIN. Verify with `EXPLAIN ANALYZE` before and after.

## Lock-wait hotspots

### [INFO] No blocking or stuck transactions right now

Snapshot only — re-run during peak load or an incident.

## Config smells

### [INFO] work_mem at default (4 MB)

Many analytical queries spill to disk at this level. Bump per-session (`SET LOCAL work_mem = '64MB';`) for heavy reports, or globally if RAM allows. Watch out: applied per sort/hash node, not per query.

## Cache hit ratio

### [WARNING] Database cache hit ratio: 94.68%

hits=16102198, disk reads=905296. OLTP workloads usually want >99%. If you just restarted, give it time to warm up. Persistent low ratio → increase shared_buffers or add RAM.

### [WARNING] `public.orders` — 94.5% hit (901096 disk reads)

Hot table reaching for disk. Either it's the largest table in the system (expected) or it lacks an index pushing the planner into full reads.

## Replication lag

### [INFO] No replicas connected

Either there are no replicas, or none were connected at the moment of the audit. Not necessarily a problem.

## Largest relations

### [INFO] `public.orders` — 136.3 MB total

heap 101.9 MB, indexes 34.4 MB. Largest relations are where bloat and cache misses cost the most — cross-reference with the bloat and unused-index sections.

### [INFO] `public.audit_log` — 14.1 MB total

heap 13.0 MB, indexes 1.1 MB. Largest relations are where bloat and cache misses cost the most — cross-reference with the bloat and unused-index sections.

### [INFO] `public.users` — 1.2 MB total

heap 824.0 KB, indexes 328.0 KB. Largest relations are where bloat and cache misses cost the most — cross-reference with the bloat and unused-index sections.

### [INFO] `public.temp_data` — 360.0 KB total

heap 328.0 KB, indexes 0 B. Largest relations are where bloat and cache misses cost the most — cross-reference with the bloat and unused-index sections.

### [INFO] `public.sessions` — 280.0 KB total

heap 248.0 KB, indexes 0 B. Largest relations are where bloat and cache misses cost the most — cross-reference with the bloat and unused-index sections.

## Connection usage

### [INFO] 7 / 100 connections in use (7%)

Approaching max_connections risks hard "too many clients" failures. If usage is high with many idle connections, put a pooler (PgBouncer) in front or lower per-app pool sizes rather than raising max_connections (each backend costs memory).

### [INFO] 1 connection(s) in state "active"



### [INFO] 1 connection(s) in state "idle"



## Tables without a primary key

### [WARNING] `public.sessions` has no primary key

Tables without a primary key can't be replicated by logical replication (without REPLICA IDENTITY FULL), are awkward for ORMs and row-level updates, and make duplicate rows hard to detect. Add a primary key or a UNIQUE NOT NULL identity column.

### [WARNING] `public.temp_data` has no primary key

Tables without a primary key can't be replicated by logical replication (without REPLICA IDENTITY FULL), are awkward for ORMs and row-level updates, and make duplicate rows hard to detect. Add a primary key or a UNIQUE NOT NULL identity column.

---
Want a senior backend engineer to read this report and send back a
prioritized fix plan with rollout steps? $800 flat, 48-hour turnaround.
See README for details.
