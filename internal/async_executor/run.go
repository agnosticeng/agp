package async_executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/agnosticeng/agp/internal/async_executor/queries"
	"github.com/agnosticeng/agp/internal/backend"
	"github.com/jackc/pgx/v5"
)

type RunOptions struct {
	Tier                 string
	MaxHeartbeatInterval time.Duration
}

func (aex *AsyncExecutor) Run(
	ctx context.Context,
	identity string,
	bkd backend.Backend,
	opts RunOptions,
) (bool, error) {
	if opts.MaxHeartbeatInterval == 0 {
		opts.MaxHeartbeatInterval = time.Second * 10
	}

	ex, err := aex.pickExecution(ctx, opts.Tier, identity, opts.MaxHeartbeatInterval)

	if err != nil {
		return false, err
	}

	if ex == nil {
		return false, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	bkdRes, err := bkd.ExecuteQuery(ctx, ex.Query, backend.WithProgressHandler(func(p backend.Progress) {
		ex, err := aex.heartbeatExecution(ctx, ex.Id, identity, opts.MaxHeartbeatInterval, p)

		if err != nil || ex.Status != StatusRunning {
			if err != nil {
				aex.logger.Error(err.Error())
			}

			cancel()
		}
	}))

	if err != nil {
		return true, aex.completeExecution(ctx, ex.Id, identity, StatusFailed, nil, err.Error())
	}

	js, err := aex.processResult(ctx, bkdRes, ex)

	if err != nil {
		return true, aex.completeExecution(ctx, ex.Id, identity, StatusFailed, nil, err.Error())
	}

	return true, aex.completeExecution(ctx, ex.Id, identity, StatusSucceeded, js, "")
}

func (aex *AsyncExecutor) processResult(
	ctx context.Context,
	bkdRes *backend.Result,
	ex *Execution,
) (json.RawMessage, error) {
	resUrl, path, err := aex.buildResultURL(ex)

	if err != nil {
		return nil, err
	}

	w, err := aex.os.Writer(ctx, resUrl)

	if err != nil {
		return nil, err
	}

	cw, err := Compressor(aex.conf.ResultStorageCompression, w)

	if err != nil {
		return nil, err
	}

	if err := json.NewEncoder(cw).Encode(bkdRes); err != nil {
		return nil, err
	}

	if err := cw.Close(); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	var md ResultMetadata

	md.NumRows = bkdRes.Rows
	md.Schema = bkdRes.Meta
	md.StoragePath = path
	md.StorageCompression = aex.conf.ResultStorageCompression

	return json.Marshal(md)
}

func (aex *AsyncExecutor) pickExecution(
	ctx context.Context,
	tier string,
	identity string,
	maxHeartbeatInterval time.Duration,
) (*Execution, error) {
	rows, err := queries.Query(ctx, aex.pool, "pick.sql", pgx.StrictNamedArgs{
		"tier":                   tier,
		"picked_by":              identity,
		"max_heartbeat_interval": maxHeartbeatInterval,
	})

	if err != nil {
		return nil, err
	}

	ex, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Execution])

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return &ex, nil
}

func (aex *AsyncExecutor) heartbeatExecution(
	ctx context.Context,
	id int64,
	identity string,
	maxHeartbeatInterval time.Duration,
	progress backend.Progress,
) (*Execution, error) {
	js, err := json.Marshal(progress)

	rows, err := queries.Query(ctx, aex.pool, "heartbeat.sql", pgx.StrictNamedArgs{
		"id":                     id,
		"picked_by":              identity,
		"max_heartbeat_interval": maxHeartbeatInterval,
		"progress":               json.RawMessage(js),
	})

	if err != nil {
		return nil, err
	}

	ex, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Execution])

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("tried to heartbeat execution %d, but is not owner", id)
	}

	if err != nil {
		return nil, err
	}

	return &ex, nil
}

func (aex *AsyncExecutor) completeExecution(
	ctx context.Context,
	id int64,
	identity string,
	status Status,
	res json.RawMessage,
	errorStr string,
) error {
	rows, err := queries.Query(ctx, aex.pool, "complete.sql", pgx.StrictNamedArgs{
		"id":        id,
		"picked_by": identity,
		"status":    status,
		"result":    res,
		"error":     errorStr,
	})

	if err != nil {
		return err
	}

	_, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Execution])

	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("tried to complete execution %d, but is not owner", id)
	}

	if err != nil {
		return err
	}

	return nil
}
