package collector

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type poolIface = *pgxpool.Pool

func collectMeta(ctx context.Context, pool poolIface, f *Findings) error {
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&f.DatabaseName); err != nil {
		return err
	}
	if err := pool.QueryRow(ctx, `SHOW server_version`).Scan(&f.PgVersion); err != nil {
		return err
	}
	return nil
}

// stubCheck returns a placeholder check that records a TODO finding.
// Replace each call site with the real implementation as it's built.
func stubCheck(id string) func(ctx context.Context, pool poolIface) ([]Finding, error) {
	return func(ctx context.Context, pool poolIface) ([]Finding, error) {
		return []Finding{{
			Severity: Info,
			Title:    "Not implemented yet",
			Detail:   "Check `" + id + "` is scaffolded but not implemented in v0.1.",
		}}, nil
	}
}
