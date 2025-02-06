package bookkeeper

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/agnosticeng/agp/internal/async_executor"
	"github.com/agnosticeng/agp/internal/process/bookkeeper"
	"github.com/agnosticeng/cnf"
	"github.com/agnosticeng/cnf/providers/env"
	"github.com/agnosticeng/cnf/providers/file"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

type config struct {
	async_executor.AsyncExecutorConfig
	bookkeeper.BookkeeperConfig
}

func Command() *cli.Command {
	return &cli.Command{
		Name: "bookkeeper",
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

			aex, err := async_executor.NewAsyncExecutor(sigctx, cfg.AsyncExecutorConfig)

			if err != nil {
				return err
			}

			defer aex.Close()
			return bookkeeper.Bookkeeper(sigctx, aex, identity.String(), cfg.BookkeeperConfig)
		},
	}
}
