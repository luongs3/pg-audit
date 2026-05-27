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
