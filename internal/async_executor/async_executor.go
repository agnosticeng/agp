package async_executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"

	"github.com/agnosticeng/agp/internal/async_executor/queries"
	"github.com/agnosticeng/agp/internal/query_hasher"
	"github.com/agnosticeng/objstr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	slogpgx "github.com/mcosta74/pgx-slog"
	"github.com/samber/lo"
	slogctx "github.com/veqryn/slog-context"
)

type AsyncExecutorConfig struct {
	Dsn                      string
	ResultStoragePrefix      string
	ResultStorageCompression ResultCompression
}

type AsyncExecutor struct {
	conf        AsyncExecutorConfig
	logger      *slog.Logger
	pool        *pgxpool.Pool
	os          *objstr.ObjectStore
	queryHasher query_hasher.QueryHasher
}

func NewAsyncExecutor(
	ctx context.Context,
	queryHasher query_hasher.QueryHasher,
	conf AsyncExecutorConfig,
) (*AsyncExecutor, error) {
	var logger = slogctx.FromCtx(ctx)

	if queryHasher == nil {
		return nil, fmt.Errorf("a query hasher must be provider")
	}

	_, err := url.Parse(conf.ResultStoragePrefix)

	if err != nil {
		return nil, fmt.Errorf("invalid result storage prefix: %w", err)
	}

	connConfig, err := pgxpool.ParseConfig(conf.Dsn)

	if err != nil {
		return nil, err
	}

	connConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   slogpgx.NewLogger(logger),
		LogLevel: tracelog.LogLevelError,
	}

	pool, err := pgxpool.NewWithConfig(ctx, connConfig)

	if err != nil {
		return nil, err
	}

	return &AsyncExecutor{
		conf:        conf,
		logger:      logger,
		pool:        pool,
		os:          objstr.FromContext(ctx),
		queryHasher: queryHasher,
	}, nil
}

func (aex *AsyncExecutor) GetQueryHasher() query_hasher.QueryHasher {
	return aex.queryHasher
}

func (aex *AsyncExecutor) GetById(ctx context.Context, id int64) (*Execution, error) {
	rows, err := queries.Query(ctx, aex.pool, "get_by_id.sql", pgx.NamedArgs{"id": id})

	if err != nil {
		return nil, err
	}

	ex, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Execution])

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &ex, nil
}

func (aex *AsyncExecutor) GetResultReader(ctx context.Context, ex *Execution) (io.ReadCloser, error) {
	if ex == nil || len(ex.Result.StoragePath) == 0 {
		return nil, nil
	}

	resUrl, _, err := aex.buildResultURL(ex)

	if err != nil {
		return nil, err
	}

	return aex.os.Reader(ctx, resUrl)
}

type ListByQueryIdOptions struct {
	Statuses  []Status
	SortBy    SortBy
	Limit     int
	QueryHash string
}

func (aex *AsyncExecutor) ListByQueryId(ctx context.Context, queryId string, opts ListByQueryIdOptions) ([]*Execution, error) {
	if len(queryId) == 0 {
		return nil, fmt.Errorf("query_id must not be empty")
	}

	if len(opts.Statuses) == 0 {
		opts.Statuses = []Status{StatusSucceeded}
	}

	if len(opts.SortBy) == 0 {
		opts.SortBy = SortByCompletedAt
	}

	if opts.Limit <= 0 {
		opts.Limit = 3
	} else {
		opts.Limit = max(opts.Limit, 100)
	}

	rows, err := queries.Query(ctx, aex.pool, "list_by_query_id.sql", pgx.NamedArgs{
		"query_id":   queryId,
		"statuses":   opts.Statuses,
		"sort_by":    opts.SortBy,
		"limit":      opts.Limit,
		"query_hash": opts.QueryHash,
	})

	if err != nil {
		return nil, err
	}

	exs, err := pgx.CollectRows(rows, pgx.RowToStructByName[Execution])

	if err != nil {
		return nil, err
	}

	return lo.ToSlicePtr(exs), nil
}

type CreateOptions struct {
	QueryId             string
	Tier                string
	CancelOtherVersions bool
}

func (aex *AsyncExecutor) Create(
	ctx context.Context,
	identity string,
	query string,
	opts CreateOptions,
) (*Execution, error) {
	if len(query) == 0 {
		return nil, fmt.Errorf("query must not be empty")
	}

	var queryHash = aex.queryHasher(query)

	if len(opts.QueryId) == 0 {
		opts.QueryId = queryHash
	}

	var tx, err = aex.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.Background())

	_, err = queries.Exec(ctx, tx, "cancel_other_versions_advisory_lock.sql", pgx.NamedArgs{
		"key": fnv1aHashInt64Sum(opts.QueryId),
	})

	if err != nil {
		return nil, err
	}

	if opts.CancelOtherVersions {
		_, err = queries.Exec(ctx, tx, "cancel_other_versions.sql", pgx.NamedArgs{
			"query_id":   opts.QueryId,
			"query_hash": queryHash,
		})

		if err != nil {
			return nil, err
		}
	}

	rows, err := queries.Query(ctx, tx, "create.sql", pgx.NamedArgs{
		"created_by": identity,
		"query_id":   opts.QueryId,
		"query_hash": queryHash,
		"query":      query,
		"tier":       opts.Tier,
	})

	if err != nil {
		return nil, err
	}

	ex, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Execution])

	if err != nil {
		return nil, err
	}

	return &ex, tx.Commit(context.Background())
}

func (aex *AsyncExecutor) Close() error {
	aex.pool.Close()
	return nil
}
