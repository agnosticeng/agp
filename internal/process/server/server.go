package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/agnosticeng/agp/internal/api/v1/async"
	"github.com/agnosticeng/agp/internal/api/v1/chproxy"
	"github.com/agnosticeng/agp/internal/api/v1/sync"
	"github.com/agnosticeng/agp/internal/async_executor"
	backend_impl "github.com/agnosticeng/agp/internal/backend/impl"
	"github.com/agnosticeng/agp/internal/signer"
	"github.com/agnosticeng/agp/pkg/client_ip_middleware"
	"github.com/agnosticeng/agp/pkg/openapi3_auth"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	oapi_middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/rs/cors"
	"github.com/samber/lo"
	"github.com/swaggest/swgui/v5emb"
	slogctx "github.com/veqryn/slog-context"
)

type AsyncAPIConfig struct {
	Enable bool
}

type SyncAPIConfig struct {
	Enable   bool
	Backends []BackendTierConfig
}

type CHProxyAPIConfig struct {
	Enable   bool
	Backends []BackendTierConfig
}

type APIConfig struct {
	Sync    SyncAPIConfig
	Async   AsyncAPIConfig
	ChProxy CHProxyAPIConfig
}

type BackendTierConfig struct {
	Tier string
	Dsn  string
}

type TLSConfig struct {
	Cert string
	Key  string
}

type ServerConfig struct {
	Addr        string
	Secret      string
	Api         APIConfig
	Tls         *TLSConfig
	DisableCors bool
	DisableGzip bool
}

func Server(ctx context.Context, aex *async_executor.AsyncExecutor, conf ServerConfig) error {
	var (
		logger = slogctx.FromCtx(ctx)
		mux    = http.NewServeMux()
	)

	var sig = signer.HMAC256Signer([]byte(conf.Secret))

	if conf.Api.Async.Enable {
		if aex == nil {
			return fmt.Errorf("AsyncExecutor must be provided for async API to work")
		}

		var validationMiddleware = validationMiddleware(
			swaggerWithServer(lo.Must(async.GetSwagger()), "/v1/async"),
			openapi3_auth.OpenAPI3Secret(
				conf.Secret,
				openapi3_auth.OpenAPI3SecretConfig{},
			),
		)
		var strictHandler = async.NewStrictHandler(async.NewServer(ctx, sig, aex), nil)
		var handler = async.HandlerWithOptions(strictHandler, async.StdHTTPServerOptions{BaseURL: "/v1/async"})
		handler = validationMiddleware(handler)
		handler = client_ip_middleware.ClientIP(handler)
		mux.Handle("/v1/async/spec.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(lo.Must(async.GetSwagger()))
		}))
		mux.Handle("/v1/async/docs/", v5emb.New("AGP Sync API", "/v1/async/spec.json", "/v1/async/docs/"))
		mux.Handle("/v1/async/", handler)
	}

	if conf.Api.Sync.Enable {
		var bkds []sync.BackendTier

		for _, backend := range conf.Api.Sync.Backends {
			bkd, err := backend_impl.NewBackend(ctx, backend.Dsn)

			if err != nil {
				return fmt.Errorf("failed to create backend for tier %s: %w", backend.Tier, err)
			}

			defer bkd.Close()
			bkds = append(bkds, sync.BackendTier{Tier: backend.Tier, Backend: bkd})
		}

		var validationMiddleware = validationMiddleware(
			swaggerWithServer(lo.Must(async.GetSwagger()), "/v1/sync"),
			openapi3_auth.OpenAPI3Secret(
				conf.Secret,
				openapi3_auth.OpenAPI3SecretConfig{
					AllowEmpty: true,
				},
			),
		)

		var server, err = sync.NewServer(ctx, bkds)

		if err != nil {
			return err
		}

		var strictHandler = sync.NewStrictHandler(server, nil)
		var handler = sync.HandlerWithOptions(strictHandler, sync.StdHTTPServerOptions{BaseURL: "/v1/sync"})
		handler = validationMiddleware(handler)
		handler = client_ip_middleware.ClientIP(handler)
		mux.Handle("/v1/sync/spec.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(lo.Must(sync.GetSwagger())) }))
		mux.Handle("/v1/sync/docs/", v5emb.New("AGP Sync API", "/v1/sync/spec.json", "/v1/sync/docs/"))
		mux.Handle("/v1/sync/", handler)
	}

	if conf.Api.ChProxy.Enable {
		var bkds []chproxy.BackendTier

		for _, backend := range conf.Api.ChProxy.Backends {
			bkds = append(bkds, chproxy.BackendTier{Tier: backend.Tier, Backend: backend.Dsn})
		}

		var validationMiddleware = validationMiddleware(
			swaggerWithServer(lo.Must(chproxy.GetSwagger()), "/v1/chproxy"),
			openapi3_auth.OpenAPI3Secret(
				conf.Secret,
				openapi3_auth.OpenAPI3SecretConfig{
					AllowEmpty: true,
				},
			),
		)

		var server, err = chproxy.NewServer(ctx, bkds)

		if err != nil {
			return err
		}

		var handler = chproxy.HandlerWithOptions(server, chproxy.StdHTTPServerOptions{BaseURL: "/v1/chproxy"})
		handler = validationMiddleware(handler)
		handler = client_ip_middleware.ClientIP(handler)
		mux.Handle("/v1/chproxy/spec.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(lo.Must(chproxy.GetSwagger())) }))
		mux.Handle("/v1/chproxy/docs/", v5emb.New("AGP Clickhouse Proxy API", "/v1/chproxy/spec.json", "/v1/chproxy/docs/"))
		mux.Handle("/v1/chproxy/", handler)
	}

	if len(conf.Addr) == 0 {
		conf.Addr = "0.0.0.0:8888"
	}

	var h http.Handler = mux

	if !conf.DisableGzip {
		h = gziphandler.GzipHandler(h)
	}

	if !conf.DisableCors {
		h = cors.Default().Handler(h)
	}

	var httpServer = http.Server{
		Addr:    conf.Addr,
		Handler: h,
	}

	if conf.Tls != nil {
		var tlsConf tls.Config

		keypair, err := tls.LoadX509KeyPair(conf.Tls.Cert, conf.Tls.Key)

		if err != nil {
			return err
		}

		tlsConf.Certificates = []tls.Certificate{keypair}
		httpServer.TLSConfig = &tlsConf
	}

	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
	}()

	if conf.Tls == nil {
		logger.Info("server running", "addr", conf.Addr, "protocol", "HTTP")
		return httpServer.ListenAndServe()
	} else {
		logger.Info("server running", "addr", conf.Addr, "protocol", "HTTPS")
		return httpServer.ListenAndServeTLS("", "")
	}
}

func validationMiddleware(
	swagger *openapi3.T,
	auth openapi3filter.AuthenticationFunc,
) func(next http.Handler) http.Handler {
	return oapi_middleware.OapiRequestValidatorWithOptions(
		swagger,
		&oapi_middleware.Options{
			SilenceServersWarning: true,
			Options: openapi3filter.Options{
				AuthenticationFunc: auth,
			},
		},
	)
}

func swaggerWithServer(swagger *openapi3.T, server string) *openapi3.T {
	swagger.Servers = openapi3.Servers{&openapi3.Server{URL: server}}
	return swagger
}
