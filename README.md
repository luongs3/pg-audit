# pg-audit

One-command Postgres performance + health audit. Run it against a database
connection string, get a Markdown report covering slow queries, missing
indexes, table/index bloat, vacuum health, lock contention, and stat-reset
advice.

```bash
go install github.com/luongs3/pg-audit/cmd/pg-audit@latest
pg-audit -dsn "postgres://localhost/mydb" -out report.md
```

## Why

`pg_stat_statements` and `pganalyze`-style insights exist, but live behind
hosted dashboards ([pganalyze](https://pganalyze.com/) $149/mo,
[pgmustard](https://www.pgmustard.com/) €95/yr) or require manual psql
incantations. `pg-audit` runs the standard playbook locally, ad-hoc, no
setup.

## Roadmap (early — open to issues)

- [x] v0.0: connect + server info
- [ ] v0.1: top-N slow queries from `pg_stat_statements`
- [ ] v0.2: missing-index hints from `pg_stat_user_tables`
- [ ] v0.3: bloat scan
- [ ] v0.4: vacuum / autovacuum health
- [ ] v0.5: Markdown report
- [ ] v0.6: HTML report

## Status

Pre-release. The `go install` path will resolve once v0.1 is tagged.
Watch the repo if you'd like a ping when that happens.

## License

MIT
