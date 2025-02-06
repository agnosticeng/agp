package impl

import (
	"context"
	"fmt"
	"net/url"

	"github.com/agnosticeng/agp/internal/backend"
	"github.com/agnosticeng/agp/internal/backend/impl/clickhouse"
)

func NewBackend(ctx context.Context, dsn string) (backend.Backend, error) {
	u, err := url.Parse(dsn)

	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "clickhouse":
		return clickhouse.NewClickhouseBackend(ctx, dsn)
	default:
		return nil, fmt.Errorf("unknwon backend scheme: %s", u.Scheme)
	}
}
