package chproxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/agnosticeng/agp/internal/utils"
	"github.com/agnosticeng/agp/pkg/client_ip_middleware"
	"github.com/samber/lo"
	slogctx "github.com/veqryn/slog-context"
)

var forwardHeaderPrefixes = []string{
	"content-type",
	"x-clickhouse-",
}

type BackendTier struct {
	Tier    string
	Backend string
}

type Server struct {
	logger *slog.Logger
	bkds   []BackendTier
	client *http.Client
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
		client: &http.Client{},
	}, nil
}

func (srv *Server) Post(w http.ResponseWriter, r *http.Request, params PostParams) {
	var (
		quotaKey string
		tier     string
		clientIp = client_ip_middleware.FromContext(r.Context())
	)

	if params.Authorization != nil {
		quotaKey = utils.DerefOr(params.QuotaKey, clientIp)
		tier = utils.Deref(params.Tier)
	} else {
		quotaKey = clientIp
	}

	srv.logger.Debug(r.URL.String(), "client_ip", clientIp, "tier", tier, "quota_key", quotaKey)

	var bkd, found = lo.Find(srv.bkds, func(v BackendTier) bool { return v.Tier == tier })

	if !found {
		httpError(srv.logger, w, fmt.Errorf("no backend found for tier: %s", tier), http.StatusInternalServerError)
		return
	}

	upstreamReq, err := http.NewRequest("POST", bkd.Backend, r.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var upstreamParams = upstreamReq.URL.Query()
	upstreamParams.Set("quota_key", quotaKey)
	upstreamParams.Set("default_format", utils.DerefOr(params.DefaultFormat, "TabSeparated"))
	upstreamReq.URL.RawQuery = upstreamParams.Encode()

	upstreamResp, err := srv.client.Do(upstreamReq)

	if err != nil {
		httpError(srv.logger, w, err, http.StatusInternalServerError)
		return
	}

	for k, v := range upstreamResp.Header {
		if len(v) == 0 {
			continue
		}

		for _, h := range forwardHeaderPrefixes {
			if strings.HasPrefix(strings.ToLower(k), h) {
				w.Header().Set(k, v[0])
				break
			}
		}
	}

	w.WriteHeader(upstreamResp.StatusCode)
	io.Copy(w, upstreamResp.Body)
}

func httpError(logger *slog.Logger, w http.ResponseWriter, err error, statusCode int) {
	logger.Error(err.Error(), "status_code", statusCode)
	http.Error(w, err.Error(), statusCode)
}
