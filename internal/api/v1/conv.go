package v1

import (
	"github.com/agnosticeng/agp/internal/backend"
)

func ToResult(bkdres *backend.Result) *Result {
	if bkdres == nil {
		return nil
	}

	var (
		res  Result
		meta []Column
	)

	res.Rows = &bkdres.NumRows

	for _, column := range bkdres.Schema {
		meta = append(meta, Column(column))
	}

	res.Meta = &meta
	res.Data = &bkdres.Rows
	return &res
}

func ToProgress(bkdprog *backend.Progress) *Progress {
	if bkdprog == nil {
		return nil
	}

	var (
		bytes     = int64(bkdprog.Bytes)
		elapsed   = bkdprog.Elapsed.Milliseconds()
		rows      = int64(bkdprog.Rows)
		totalRows = int64(bkdprog.TotalRows)
	)

	return &Progress{
		Bytes:     &bytes,
		Elapsed:   &elapsed,
		Rows:      &rows,
		TotalRows: &totalRows,
	}
}

func ToResultEvent(bkdres *backend.Result, err error) *ResultEvent {
	if bkdres == nil && err == nil {
		return nil
	}

	if err != nil {
		var str = err.Error()
		return &ResultEvent{Error: &str}
	}

	var res = ToResult(bkdres)

	return &ResultEvent{
		Rows: res.Rows,
		Meta: res.Meta,
		Data: res.Data,
	}
}
