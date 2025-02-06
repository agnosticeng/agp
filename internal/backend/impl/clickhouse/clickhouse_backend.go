package clickhouse

import (
	"context"
	"reflect"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/agnosticeng/agp/internal/backend"
)

type ClickhouseBackend struct {
	conn driver.Conn
}

func NewClickhouseBackend(ctx context.Context, dsn string) (*ClickhouseBackend, error) {
	chopts, err := clickhouse.ParseDSN(dsn)

	if err != nil {
		return nil, err
	}

	chconn, err := clickhouse.Open(chopts)

	if err != nil {
		return nil, err
	}

	return &ClickhouseBackend{chconn}, nil
}

func (b *ClickhouseBackend) ExecuteQuery(ctx context.Context, query string, optfns ...backend.RunOption) (*backend.Result, error) {
	var (
		runOpts = backend.BuildRunOptions(optfns...)
	)

	var p backend.Progress

	if runOpts.ProgressHandler != nil {
		ctx = clickhouse.Context(ctx, clickhouse.WithProgress(func(delta *clickhouse.Progress) {
			p.Bytes += delta.Bytes
			p.Rows += delta.Rows
			p.Elapsed += delta.Elapsed
			p.TotalRows += delta.TotalRows

			runOpts.ProgressHandler(p)
		}))
	}

	if len(runOpts.QuotaKey) > 0 {
		ctx = clickhouse.Context(ctx, clickhouse.WithQuotaKey(runOpts.QuotaKey))
	}

	queryRes, err := b.conn.Query(ctx, query)

	if err != nil {
		return nil, err
	}

	var (
		columnTypes = queryRes.ColumnTypes()
		columnNames = queryRes.Columns()
		rows        = make([]map[string]any, 0)
	)

	for queryRes.Next() {
		var values = make([]any, len(columnNames))

		for i := 0; i < len(columnNames); i++ {
			values[i] = reflect.New(columnTypes[i].ScanType()).Interface()
		}

		if err := queryRes.Scan(values...); err != nil {
			return nil, err
		}

		var row = make(map[string]any)

		for i, v := range values {
			row[columnNames[i]] = v
		}

		rows = append(rows, row)
	}

	var res = backend.Result{
		NumRows: int64(len(rows)),
		Rows:    rows,
	}

	for i := 0; i < len(columnNames); i++ {
		res.Schema = append(res.Schema, backend.Column{
			Name: columnNames[i],
			Type: columnTypes[i].DatabaseTypeName(),
		})
	}

	return &res, queryRes.Err()
}

func (b *ClickhouseBackend) Close() error {
	return b.conn.Close()
}
