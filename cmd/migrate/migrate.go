package migrate

import (
	"github.com/agnosticeng/agp/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
	"github.com/urfave/cli/v2"
)

var Flags = []cli.Flag{
	&cli.StringFlag{Name: "dsn", Value: "postgres://postgres:postgres@localhost:5432/postgres?sslmode=allow"},
	&cli.StringFlag{Name: "version-table", Value: "agp_schema_version"},
}

func Command() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Flags: Flags,
		Action: func(ctx *cli.Context) error {
			var (
				dsn          = ctx.String("dsn")
				versionTable = ctx.String("version-table")
			)

			conf, err := pgx.ParseConfig(dsn)

			if err != nil {
				return err
			}

			conn, err := pgx.ConnectConfig(ctx.Context, conf)

			if err != nil {
				return err
			}

			m, err := migrate.NewMigrator(ctx.Context, conn, versionTable)

			if err != nil {
				return err
			}

			if err := m.LoadMigrations(migrations.FS); err != nil {
				return err
			}

			return m.Migrate(ctx.Context)
		},
	}
}
