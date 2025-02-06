package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/agnosticeng/agp/cmd/bookkeeper"
	"github.com/agnosticeng/agp/cmd/migrate"
	"github.com/agnosticeng/agp/cmd/server"
	"github.com/agnosticeng/agp/cmd/standalone"
	"github.com/agnosticeng/agp/cmd/worker"
	"github.com/agnosticeng/cliutils"
	"github.com/agnosticeng/cnf"
	"github.com/agnosticeng/cnf/providers/env"
	objstrcli "github.com/agnosticeng/objstr/cli"
	"github.com/agnosticeng/panicsafe"
	"github.com/agnosticeng/slogcli"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.App{
		Name:  "agp",
		Flags: slogcli.SlogFlags(),
		Before: cliutils.CombineBeforeFuncs(
			slogcli.SlogBefore,
			objstrcli.ObjStrBefore(cnf.WithProvider(env.NewEnvProvider("OBJSTR"))),
		),
		After: cliutils.CombineAfterFuncs(
			objstrcli.ObjStrAfter,
			slogcli.SlogAfter,
		),
		Commands: []*cli.Command{
			migrate.Command(),
			worker.Command(),
			bookkeeper.Command(),
			server.Command(),
			standalone.Command(),
		},
	}

	var err = panicsafe.Recover(func() error { return app.Run(os.Args) })

	if err != nil {
		slog.Error(fmt.Sprintf("%v", err))
		os.Exit(1)
	}
}
