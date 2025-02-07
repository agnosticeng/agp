package standalone

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/agnosticeng/agp/internal/async_executor"
	"github.com/agnosticeng/agp/internal/process/bookkeeper"
	"github.com/agnosticeng/agp/internal/process/server"
	"github.com/agnosticeng/agp/internal/process/worker"
	"github.com/agnosticeng/agp/internal/query_hasher"
	"github.com/agnosticeng/agp/migrations"
	"github.com/agnosticeng/cnf"
	"github.com/agnosticeng/cnf/providers/env"
	"github.com/agnosticeng/cnf/providers/file"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

type config struct {
	async_executor.AsyncExecutorConfig
	Worker     worker.WorkerConfig
	Server     server.ServerConfig
	Bookkeeper bookkeeper.BookkeeperConfig
}

func Command() *cli.Command {
	return &cli.Command{
		Name: "standalone",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "migrate"},
			&cli.StringFlag{Name: "version-table", Value: "agp_schema_version"},
		},
		Action: func(ctx *cli.Context) error {
			var (
				sigctx, sigctxcancel = signal.NotifyContext(ctx.Context, os.Interrupt, syscall.SIGTERM)
				identity             = uuid.Must(uuid.NewV7())
				cfg                  config
				cfgOpts              = []cnf.OptionFunc{
					cnf.WithProvider(env.NewEnvProvider("AGP")),
				}
			)

			defer sigctxcancel()

			if ctx.Args().Len() > 0 {
				cfgOpts = append(cfgOpts, cnf.WithProvider(file.NewFileProvider(ctx.Args().Get(0))))
			}

			if err := cnf.Load(&cfg, cfgOpts...); err != nil {
				return err
			}

			if ctx.Bool("migrate") {
				if err := migrations.Migrate(
					ctx.Context,
					cfg.AsyncExecutorConfig.Dsn,
					ctx.String("version-table"),
				); err != nil {
					return err
				}
			}

			aex, err := async_executor.NewAsyncExecutor(sigctx, query_hasher.SHA256QueryHasher, cfg.AsyncExecutorConfig)

			if err != nil {
				return err
			}

			defer aex.Close()

			var group, groupCtx = errgroup.WithContext(sigctx)

			group.Go(func() error { return server.Server(groupCtx, aex, cfg.Server) })
			group.Go(func() error { return worker.Worker(groupCtx, aex, identity.String(), cfg.Worker) })
			group.Go(func() error { return bookkeeper.Bookkeeper(groupCtx, aex, identity.String(), cfg.Bookkeeper) })

			return group.Wait()
		},
	}
}
