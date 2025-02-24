package bookkeeper

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/agnosticeng/agp/cmd/bookkeeper/gc_cleanup"
	"github.com/agnosticeng/agp/cmd/bookkeeper/gc_mark"
	"github.com/agnosticeng/agp/internal/async_executor"
	"github.com/agnosticeng/agp/internal/process/bookkeeper"
	"github.com/agnosticeng/agp/internal/query_hasher"
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
		Subcommands: []*cli.Command{
			gc_mark.Command(),
			gc_cleanup.Command(),
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

			aex, err := async_executor.NewAsyncExecutor(sigctx, query_hasher.SHA256QueryHasher, cfg.AsyncExecutorConfig)

			if err != nil {
				return err
			}

			defer aex.Close()
			return bookkeeper.Bookkeeper(sigctx, aex, identity.String(), cfg.BookkeeperConfig)
		},
	}
}
