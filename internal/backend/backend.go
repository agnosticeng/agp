package backend

import (
	"context"
	"time"
)

type RunOptions struct {
	QuotaKey        string
	ProgressHandler func(Progress)
}

type RunOption func(*RunOptions)

func BuildRunOptions(fns ...RunOption) *RunOptions {
	var opts RunOptions

	for _, fn := range fns {
		fn(&opts)
	}

	return &opts
}

type Progress struct {
	Rows      uint64        `json:"rows"`
	Bytes     uint64        `json:"bytes"`
	TotalRows uint64        `json:"total_rows"`
	Elapsed   time.Duration `json:"elapsed"`
}

func WithProgressHandler(h func(Progress)) RunOption {
	return func(ro *RunOptions) {
		ro.ProgressHandler = h
	}
}

func WithQuotaKey(key string) RunOption {
	return func(ro *RunOptions) {
		ro.QuotaKey = key
	}
}

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Schema []Column

type Result struct {
	Schema  Schema           `json:"schema"`
	NumRows int64            `json:"num_rows"`
	Rows    []map[string]any `json:"rows"`
}

type Backend interface {
	ExecuteQuery(ctx context.Context, query string, opts ...RunOption) (*Result, error)
	Close() error
}

type BackendFactory func(context.Context, string) (Backend, error)
