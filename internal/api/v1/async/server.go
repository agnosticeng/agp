package async

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/agnosticeng/agp/internal/async_executor"
	"github.com/agnosticeng/agp/internal/signer"
	"github.com/agnosticeng/agp/internal/utils"
	"github.com/agnosticeng/agp/pkg/client_ip_middleware"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"
	slogctx "github.com/veqryn/slog-context"
)

type Server struct {
	logger *slog.Logger
	signer signer.Signer
	aex    *async_executor.AsyncExecutor
}

func NewServer(
	ctx context.Context,
	signer signer.Signer,
	aex *async_executor.AsyncExecutor,
) *Server {
	return &Server{
		logger: slogctx.FromCtx(ctx),
		signer: signer,
		aex:    aex,
	}
}

func (srv *Server) PostExecutions(
	ctx context.Context,
	request PostExecutionsRequestObject,
) (PostExecutionsResponseObject, error) {
	var (
		sql     string
		secrets = make(map[string]string)
	)

	if request.TextBody != nil {
		sql = *request.TextBody
	} else {
		sql = request.JSONBody.Sql

		if request.JSONBody.Secrets != nil {
			for _, secret := range *request.JSONBody.Secrets {
				secrets[secret.Key] = secret.Value
			}
		}
	}

	ex, err := srv.aex.Create(
		ctx,
		utils.DerefOr(request.Params.QuotaKey, client_ip_middleware.FromContext(ctx)),
		sql,
		async_executor.CreateOptions{
			QueryId: utils.Deref(request.Params.QueryId),
			Tier:    utils.Deref(request.Params.Tier),
			Secrets: secrets,
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
	var sig = srv.signer([]byte(fmt.Sprintf("%d%d", request.ExecutionId, request.Params.Expiration)))

	if hex.EncodeToString(sig) != request.Params.Signature {
		return nil, fmt.Errorf("invalid signature")
	}

	if time.Unix(request.Params.Expiration, 0).Before(time.Now()) {
		return nil, fmt.Errorf("signed url is expired")
	}

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

func (srv *Server) PostSearch(ctx context.Context, request PostSearchRequestObject) (PostSearchResponseObject, error) {
	if request.Body == nil || len(*request.Body) == 0 {
		return PostSearch200JSONResponse{}, nil
	}

	var p = pool.
		NewWithResults[[]Execution]().
		WithErrors().
		WithContext(ctx).
		WithCancelOnError().
		WithMaxGoroutines(3)

	for _, item := range *request.Body {
		var opts = async_executor.ListByQueryIdOptions{
			QueryHash: utils.Deref(item.QueryHash),
			Limit:     int(utils.Deref(item.Limit)),
			SortBy:    async_executor.SortBy(utils.Deref(item.SortBy)),
			Statuses: lo.Map(
				utils.Deref(item.Statuses),
				func(st ExecutionStatus, _ int) async_executor.Status {
					return async_executor.Status(st)
				},
			),
		}

		p.Go(func(ctx context.Context) ([]Execution, error) {
			exs, err := srv.aex.ListByQueryId(ctx, item.QueryId, opts)

			if err != nil {
				return nil, err
			}

			return lo.Map(exs, func(ex *async_executor.Execution, _ int) Execution {
				return *ToExecution(ex)
			}), nil
		})
	}

	res, err := p.Wait()

	if err != nil {
		return nil, err
	}

	return PostSearch200JSONResponse(res), nil
}
