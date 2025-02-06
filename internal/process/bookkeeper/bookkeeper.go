package bookkeeper

import (
	"context"
	"log/slog"
	"time"

	"github.com/agnosticeng/agp/internal/async_executor"
	slogctx "github.com/veqryn/slog-context"
	"golang.org/x/sync/errgroup"
)

type BookkeeperConfig struct {
	FailDeadInterval time.Duration
}

func Bookkeeper(ctx context.Context, aex *async_executor.AsyncExecutor, identity string, conf BookkeeperConfig) error {
	var (
		group, groupctx = errgroup.WithContext(ctx)
		logger          = slogctx.FromCtx(ctx)
	)

	if conf.FailDeadInterval == 0 {
		conf.FailDeadInterval = time.Second * 10
	}

	group.Go(func() error {
		return loop(
			groupctx,
			logger.With("loop", "FAIL_DEAD"),
			conf.FailDeadInterval,
			func() (bool, error) {
				var ctx, cancel = context.WithTimeout(groupctx, conf.FailDeadInterval)
				defer cancel()

				return aex.FailDead(
					ctx,
					identity,
					async_executor.FailDeadOptions{
						LeaseDuration: conf.FailDeadInterval * 2,
					},
				)
			},
		)
	})

	return group.Wait()
}

func loop(
	ctx context.Context,
	logger *slog.Logger,
	pollInterval time.Duration,
	f func() (bool, error),
) error {
	var nextPollInterval time.Duration
	logger.Info("starting loop")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(nextPollInterval):
			var t0 = time.Now()
			run, err := f()

			if !run {
				logger.Debug("not leader, nothing has been run")
				nextPollInterval = pollInterval
				continue
			}

			if err != nil {
				return err
			}

			logger.Debug("loop completed", "duration", time.Since(t0))
			nextPollInterval = pollInterval
		}
	}
}
