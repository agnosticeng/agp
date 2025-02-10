package async_executor

import (
	"context"
	"time"

	"github.com/agnosticeng/agp/internal/async_executor/queries"
	"github.com/jackc/pgx/v5"
	"github.com/sourcegraph/conc/iter"
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

type GCMarkOptions struct {
	LeaseDuration       time.Duration
	Limit               int
	CanceledExpiration  time.Duration
	FailedExpiration    time.Duration
	SucceededExpiration time.Duration
}

func (aex *AsyncExecutor) GCMark(ctx context.Context, identity string, opts GCMarkOptions) (bool, error) {
	if opts.LeaseDuration == 0 {
		opts.LeaseDuration = time.Second * 10
	}

	if opts.Limit <= 0 {
		opts.Limit = 1000
	}

	if opts.CanceledExpiration == 0 {
		opts.CanceledExpiration = time.Hour
	}

	if opts.FailedExpiration == 0 {
		opts.FailedExpiration = time.Hour * 168
	}

	if opts.SucceededExpiration == 0 {
		opts.SucceededExpiration = time.Hour * 24 * 90
	}

	return aex.withLease(
		ctx,
		"GC_MARK",
		identity,
		opts.LeaseDuration,
		func() error {
			rows, err := queries.Query(ctx, aex.pool, "gc_mark_expired.sql", pgx.NamedArgs{
				"limit":                opts.Limit,
				"canceled_expiration":  opts.CanceledExpiration,
				"failed_expiration":    opts.FailedExpiration,
				"succeeded_expiration": opts.SucceededExpiration,
			})

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

type GCCleanupOptions struct {
	LeaseDuration time.Duration
	Limit         int
}

func (aex *AsyncExecutor) GCCleanup(ctx context.Context, identity string, opts GCCleanupOptions) (bool, error) {
	if opts.LeaseDuration == 0 {
		opts.LeaseDuration = time.Second * 10
	}

	if opts.Limit <= 0 {
		opts.Limit = 1000
	}

	return aex.withLease(
		ctx,
		"GC_CLEANUP",
		identity,
		opts.LeaseDuration,
		func() error {
			rows, err := queries.Query(ctx, aex.pool, "gc_list_expired.sql", pgx.NamedArgs{
				"limit": opts.Limit,
			})

			if err != nil {
				return err
			}

			exs, err := pgx.CollectRows(rows, pgx.RowToStructByName[Execution])

			if err != nil {
				return err
			}

			ids, err := iter.MapErr(exs, func(ex *Execution) (int64, error) {
				if ex.Result == nil || len(ex.Result.StoragePath) == 0 {
					return ex.Id, nil
				}

				u, _, err := aex.buildResultURL(ex)

				if err != nil {
					return 0, err
				}

				return ex.Id, aex.os.Delete(ctx, u)
			})

			if err != nil {
				return err
			}

			rows, err = queries.Query(ctx, aex.pool, "gc_delete_by_id.sql", pgx.NamedArgs{
				"ids": ids,
			})

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
