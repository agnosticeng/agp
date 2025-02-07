package server

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/agnosticeng/agp/internal/async_executor"
	"github.com/agnosticeng/agp/internal/process/server"
	"github.com/agnosticeng/agp/internal/query_hasher"
	"github.com/agnosticeng/cnf"
	"github.com/agnosticeng/cnf/providers/env"
	"github.com/agnosticeng/cnf/providers/file"
	"github.com/urfave/cli/v2"
)

type config struct {
	async_executor.AsyncExecutorConfig
	server.ServerConfig
}

func Command() *cli.Command {
	return &cli.Command{
		Name: "server",
		Action: func(ctx *cli.Context) error {
			var (
				sigctx, sigctxcancel = signal.NotifyContext(ctx.Context, os.Interrupt, syscall.SIGTERM)
				cfg                  config
				cfgOpts              = []cnf.OptionFunc{
					cnf.WithProvider(env.NewEnvProvider("AGP")),
				}
				aex *async_executor.AsyncExecutor
				err error
			)

			defer sigctxcancel()

			if ctx.Args().Len() > 0 {
				cfgOpts = append(cfgOpts, cnf.WithProvider(file.NewFileProvider(ctx.Args().Get(0))))
			}

			if err := cnf.Load(&cfg, cfgOpts...); err != nil {
				return err
			}

			if len(cfg.Dsn) > 0 {
				aex, err = async_executor.NewAsyncExecutor(sigctx, query_hasher.SHA256QueryHasher, cfg.AsyncExecutorConfig)

				if err != nil {
					return err
				}

				defer aex.Close()
			}

			return server.Server(sigctx, aex, cfg.ServerConfig)
		},
	}
}
