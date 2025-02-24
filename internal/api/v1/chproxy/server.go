package chproxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/agnosticeng/agp/internal/utils"
	"github.com/agnosticeng/agp/pkg/client_ip_middleware"
	"github.com/samber/lo"
	slogctx "github.com/veqryn/slog-context"
)

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
	)

	if params.Authorization != nil {
		quotaKey = utils.DerefOr(params.QuotaKey, client_ip_middleware.FromContext(r.Context()))
		tier = utils.Deref(params.Tier)
	} else {
		quotaKey = client_ip_middleware.FromContext(r.Context())
	}

	var bkd, found = lo.Find(srv.bkds, func(v BackendTier) bool { return v.Tier == tier })

	if !found {
		http.Error(w, fmt.Sprintf("no backend found for tier: %s", tier), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range upstreamResp.Header {
		if len(v) == 0 {
			continue
		}

		w.Header().Set(k, v[0])
	}

	w.WriteHeader(upstreamResp.StatusCode)
	io.Copy(w, upstreamResp.Body)
}
