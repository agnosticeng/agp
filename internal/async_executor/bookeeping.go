package async_executor

import (
	"context"
	"time"

	"github.com/agnosticeng/agp/internal/async_executor/queries"
	slogctx "github.com/veqryn/slog-context"
)

type FailDeadOptions struct {
	LeaseDuration time.Duration
}

func (aex *AsyncExecutor) FailDead(ctx context.Context, identity string, opts FailDeadOptions) (bool, error) {
	if opts.LeaseDuration == 0 {
		opts.LeaseDuration = time.Second * 10
	}

	return aex.withLease(
		ctx,
		"FAIL_DEAD",
		identity,
		opts.LeaseDuration,
		func() error {
			rows, err := queries.Query(ctx, aex.pool, "fail_dead.sql", nil)

			if err != nil {
				return err
			}

			count, err := countRows(rows)

			if err != nil {
				return err
			}

			slogctx.FromCtx(ctx).Info("run", "count", count)
			return err
		},
	)
}
