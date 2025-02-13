package backend

import (
	"context"
	"time"
)

type RunOptions struct {
	QuotaKey        string
	ProgressHandler func(Progress)
	Parameters      map[string]string
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

func WithParameters(params map[string]string) RunOption {
	return func(ro *RunOptions) {
		ro.Parameters = params
	}
}

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Schema []Column

type Result struct {
	Meta Schema           `json:"meta"`
	Rows int64            `json:"rows"`
	Data []map[string]any `json:"data"`
}

type Backend interface {
	ExecuteQuery(ctx context.Context, query string, opts ...RunOption) (*Result, error)
	Close() error
}

type BackendFactory func(context.Context, string) (Backend, error)
