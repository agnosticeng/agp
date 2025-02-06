package async

import (
	"context"
	"log/slog"
	"time"

	"github.com/agnosticeng/agp/internal/async_executor"
	"github.com/agnosticeng/agp/internal/utils"
	"github.com/agnosticeng/agp/pkg/client_ip_middleware"
	slogctx "github.com/veqryn/slog-context"
)

type Server struct {
	logger *slog.Logger
	aex    *async_executor.AsyncExecutor
}

func NewServer(
	ctx context.Context,
	aex *async_executor.AsyncExecutor,
) *Server {
	return &Server{
		logger: slogctx.FromCtx(ctx),
		aex:    aex,
	}
}

func (srv *Server) PostExecutions(
	ctx context.Context,
	request PostExecutionsRequestObject,
) (PostExecutionsResponseObject, error) {
	ex, err := srv.aex.Create(
		ctx,
		utils.DerefOr(request.Params.QuotaKey, client_ip_middleware.FromContext(ctx)),
		utils.Deref(request.Body),
		async_executor.CreateOptions{
			QueryId: utils.Deref(request.Params.QueryId),
			Tier:    utils.Deref(request.Params.Tier),
		},
	)

	if err != nil {
		return nil, err
	}

	return PostExecutions201JSONResponse(*ToExecution(ex)), nil
}

func (srv *Server) GetExecutionsExecutionId(
	ctx context.Context,
	request GetExecutionsExecutionIdRequestObject,
) (GetExecutionsExecutionIdResponseObject, error) {
	ex, err := srv.aex.GetById(ctx, request.ExecutionId)

	if err != nil {
		return nil, err
	}

	if ex == nil {
		return GetExecutionsExecutionId404Response{}, nil

	}

	return GetExecutionsExecutionId200JSONResponse(*ToExecution(ex)), nil
}

func (srv *Server) GetExecutionsExecutionIdResult(
	ctx context.Context,
	request GetExecutionsExecutionIdResultRequestObject,
) (GetExecutionsExecutionIdResultResponseObject, error) {
	ex, err := srv.aex.GetById(ctx, request.ExecutionId)

	if err != nil {
		return nil, err
	}

	if ex == nil {
		return GetExecutionsExecutionIdResult404Response{}, nil
	}

	r, err := srv.aex.GetResultReader(ctx, ex)

	if err != nil {
		return nil, err
	}

	cr, err := async_executor.Decompressor(ex.Result.StorageCompression, r)

	if err != nil {
		return nil, err
	}

	return GetExecutionsExecutionIdResult200ApplicationoctetStreamResponse{Body: cr}, nil
}

func (srv *Server) PostRun(ctx context.Context, request PostRunRequestObject) (PostRunResponseObject, error) {
	var (
		query      = utils.Deref(request.Body)
		staleAfter = utils.Deref(request.Params.StaleAfter)
	)

	ex, err := srv.aex.ListByQueryId(
		ctx,
		utils.DerefOr(request.Params.QueryId, srv.aex.QueryId(query)),
		async_executor.ListByQueryIdOptions{
			Statuses: []async_executor.Status{async_executor.StatusSucceeded},
			SortBy:   async_executor.SortByCompletedAt,
			Limit:    1,
		},
	)

	if err != nil {
		return nil, err
	}

	if len(ex) == 0 || ex[0].CompletedAt.Add(time.Second*time.Duration(staleAfter)).Before(time.Now()) {
		_, err = srv.aex.Create(
			ctx,
			utils.DerefOr(request.Params.QuotaKey, client_ip_middleware.FromContext(ctx)),
			query,
			async_executor.CreateOptions{
				QueryId: utils.Deref(request.Params.QueryId),
				Tier:    utils.Deref(request.Params.Tier),
			},
		)

		if err != nil {
			return nil, err
		}
	}

	if len(ex) == 0 {
		return PostRun201Response{}, nil
	}

	r, err := srv.aex.GetResultReader(ctx, ex[0])

	if err != nil {
		return nil, err
	}

	return PostRun200ApplicationoctetStreamResponse{Body: r}, nil
}
