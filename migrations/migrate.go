package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
)

func Migrate(ctx context.Context, dsn string, versionTable string) error {
	conf, err := pgx.ParseConfig(dsn)

	if err != nil {
		return err
	}

	conn, err := pgx.ConnectConfig(ctx, conf)

	if err != nil {
		return err
	}

	m, err := migrate.NewMigrator(ctx, conn, versionTable)

	if err != nil {
		return err
	}

	if err := m.LoadMigrations(FS); err != nil {
		return err
	}

	return m.Migrate(ctx)
}
