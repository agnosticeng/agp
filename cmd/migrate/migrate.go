package migrate

import (
	"github.com/agnosticeng/agp/migrations"
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
			return migrations.Migrate(ctx.Context, ctx.String("dsn"), ctx.String("version-table"))
		},
	}
}
