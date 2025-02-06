package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/agnosticeng/agp/internal/async_executor"
	backend_impl "github.com/agnosticeng/agp/internal/backend/impl"
	"github.com/samber/lo"
	slogctx "github.com/veqryn/slog-context"
	"golang.org/x/sync/errgroup"
)

type BackendTier struct {
	Tier  string
	Count int
	Dsn   string
}

type WorkerConfig struct {
	PollInterval         time.Duration
	MaxHeartbeatInterval time.Duration
	Backends             []BackendTier
}

func Worker(ctx context.Context, aex *async_executor.AsyncExecutor, identity string, conf WorkerConfig) error {
	var logger = slogctx.FromCtx(ctx)

	if len(conf.Backends) == 0 {
		return fmt.Errorf("config must contains at least one backend tier")
	}

	if len(lo.FindDuplicatesBy(conf.Backends, func(bkd BackendTier) string { return bkd.Tier })) > 0 {
		return fmt.Errorf("config has duplicate tier entries")
	}

	if conf.PollInterval == 0 {
		conf.PollInterval = 1 * time.Second
	}

	if conf.MaxHeartbeatInterval == 0 {
		conf.MaxHeartbeatInterval = 10 * time.Second
	}

	logger.Info(
		"worker starting",
		"identity", identity,
		"poll_interval", conf.PollInterval,
		"max_heartbeat_interval", conf.MaxHeartbeatInterval,
	)

	var group, groupctx = errgroup.WithContext(ctx)

	for _, backend := range conf.Backends {
		if backend.Count <= 0 {
			backend.Count = 1
		}

		bkd, err := backend_impl.NewBackend(ctx, backend.Dsn)

		if err != nil {
			return err
		}

		defer bkd.Close()

		for i := 0; i < backend.Count; i++ {
			group.Go(func() error {
				var (
					nextPollInterval time.Duration
					workerId         = fmt.Sprintf("%s-%d", identity, i)
					logger           = logger.With("worker_id", workerId)
				)

				for {
					select {
					case <-groupctx.Done():
						return nil

					case <-time.After(nextPollInterval):
						run, err := aex.Run(groupctx, workerId, bkd, async_executor.RunOptions{
							MaxHeartbeatInterval: conf.MaxHeartbeatInterval,
							Tier:                 backend.Tier,
						})

						if !run {
							logger.Debug("no query to run")
							nextPollInterval = conf.PollInterval
							continue
						}

						if err != nil {
							logger.Info(
								"query completed",
								"error", err.Error(),
							)
						} else {
							logger.Info(
								"query completed",
							)
						}
					}
				}
			})
		}
	}

	return group.Wait()
}
