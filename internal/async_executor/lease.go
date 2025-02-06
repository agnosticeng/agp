package async_executor

import (
	"context"
	"fmt"
	"time"

	"github.com/agnosticeng/agp/internal/async_executor/queries"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

func (aex *AsyncExecutor) deleteLease(
	ctx context.Context,
	key string,
	owner string,
) error {
	rows, err := queries.Query(ctx, aex.pool, "delete_lease.sql", pgx.NamedArgs{
		"key":   key,
		"owner": owner,
	})

	if err != nil {
		return err
	}

	exs, err := pgx.CollectRows(rows, pgx.RowToStructByName[Lease])

	if err != nil {
		return err
	}

	if len(exs) == 0 {
		return fmt.Errorf("not longer owner of key: %s", key)
	}

	return nil
}

func (aex *AsyncExecutor) upsertLease(
	ctx context.Context,
	key string,
	owner string,
	leaseDuration time.Duration,
) (bool, error) {
	rows, err := queries.Query(ctx, aex.pool, "create_lease.sql", pgx.NamedArgs{
		"key":            key,
		"owner":          owner,
		"lease_duration": leaseDuration,
	})

	if err != nil {
		return false, err
	}

	exs, err := pgx.CollectRows(rows, pgx.RowToStructByName[Lease])

	if err != nil {
		return false, err
	}

	if len(exs) == 0 {
		return false, nil
	}

	return true, nil
}

func (aex *AsyncExecutor) withLease(
	ctx context.Context,
	key string,
	workerId string,
	leaseDuration time.Duration,
	f func() error,
) (bool, error) {
	if leaseDuration == 0 {
		leaseDuration = time.Second * 10
	}

	leader, err := aex.upsertLease(ctx, key, workerId, leaseDuration)

	if err != nil {
		return false, nil
	}

	if !leader {
		return false, nil
	}

	var (
		group, groupctx = errgroup.WithContext(ctx)
		closeChan       = make(chan struct{}, 1)
	)

	group.Go(func() error {
		var t = leaseDuration / 2

		for {
			select {
			case <-closeChan:
				return nil
			case <-groupctx.Done():
				return groupctx.Err()
			case <-time.After(t):
				if _, err := aex.upsertLease(groupctx, key, workerId, leaseDuration); err != nil {
					return err
				}
			}
		}
	})

	group.Go(func() error {
		defer close(closeChan)
		return f()
	})

	return true, group.Wait()
}
