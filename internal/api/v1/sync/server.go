package sync

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	v1 "github.com/agnosticeng/agp/internal/api/v1"
	"github.com/agnosticeng/agp/internal/backend"
	"github.com/agnosticeng/agp/internal/utils"
	"github.com/agnosticeng/agp/pkg/client_ip_middleware"
	"github.com/agnosticeng/agp/pkg/json_text_event_stream"
	"github.com/samber/lo"
	slogctx "github.com/veqryn/slog-context"
)

type BackendTier struct {
	Tier    string
	Backend backend.Backend
}

type Server struct {
	logger *slog.Logger
	bkds   []BackendTier
}

func NewServer(
	ctx context.Context,
	bkds []BackendTier,
) (*Server, error) {
	if len(bkds) == 0 {
		return nil, fmt.Errorf("at least one backend tier must be specified")
	}

	return &Server{
		logger: slogctx.FromCtx(ctx),
		bkds:   bkds,
	}, nil
}

func (srv *Server) PostRun(ctx context.Context, request PostRunRequestObject) (PostRunResponseObject, error) {
	var (
		quotaKey string
		tier     string
	)

	if request.Params.Authorization != nil {
		quotaKey = utils.DerefOr(request.Params.QuotaKey, client_ip_middleware.FromContext(ctx))
		tier = utils.Deref(request.Params.Tier)
	} else {
		quotaKey = client_ip_middleware.FromContext(ctx)
	}

	var bkd, found = lo.Find(srv.bkds, func(v BackendTier) bool { return v.Tier == tier })

	if !found {
		return nil, fmt.Errorf("no backend found for tier: %s", tier)
	}

	if !utils.DerefOr(request.Params.Stream, false) {
		res, err := bkd.Backend.ExecuteQuery(
			ctx,
			*request.Body,
			backend.WithQuotaKey(quotaKey),
		)

		if err != nil {
			return nil, err
		}

		return PostRun200JSONResponse(*(v1.ToResult(res))), nil
	}

	var (
		r, w = io.Pipe()
		enc  = json_text_event_stream.NewJSONTextEventStreamEncoder(w)
	)

	go func() {
		defer w.Close()

		res, err := bkd.Backend.ExecuteQuery(
			ctx,
			*request.Body,
			backend.WithQuotaKey(quotaKey),
			backend.WithProgressHandler(func(p backend.Progress) {
				enc.Encode("progress", v1.ProgressEvent{Progress: v1.ToProgress(&p)})
			}),
		)

		enc.Encode("result", v1.ToResultEvent(res, err))
	}()

	return PostRun200TexteventStreamResponse{Body: r}, nil
}
