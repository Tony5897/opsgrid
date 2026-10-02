package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Tony5897/opsgrid/internal/api"
	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/config"
	"github.com/Tony5897/opsgrid/internal/platform/health"
	"github.com/Tony5897/opsgrid/internal/platform/httpx"
	"github.com/Tony5897/opsgrid/internal/platform/webui"
)

// RunAPI runs the HTTP API (which also serves the SPA) until ctx is
// cancelled, then shuts down gracefully:
//
//	readiness=false → stop accepting, drain in-flight requests (bounded)
//	→ stop internal server → close Redis/PostgreSQL → flush telemetry.
func RunAPI(ctx context.Context) (err error) {
	rt, err := newRuntime(ctx, config.RoleAPI)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rt.close(ctx)) }()

	rt.health = health.New(2*time.Second, rt.baseChecks(false)...)

	public, err := rt.publicHandler()
	if err != nil {
		return err
	}
	publicSrv := httpx.NewServer(ctx, httpx.ServerOptions{
		Addr:              rt.cfg.HTTP.Addr,
		Handler:           public,
		ReadHeaderTimeout: rt.cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       rt.cfg.HTTP.ReadTimeout,
		WriteTimeout:      rt.cfg.HTTP.WriteTimeout,
		IdleTimeout:       rt.cfg.HTTP.IdleTimeout,
		Logger:            rt.log,
	})
	return rt.serveWithInternal(ctx, publicSrv)
}

// serveWithInternal runs the internal health server for the whole process
// lifetime and the public server until ctx is cancelled. Readiness flips to
// false the moment shutdown begins, so load balancers stop routing first.
func (rt *runtime) serveWithInternal(ctx context.Context, public *http.Server) error {
	internalSrv := httpx.NewServer(ctx, httpx.ServerOptions{
		Addr:              rt.cfg.HTTP.InternalAddr,
		Handler:           rt.health.Mux(),
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
		Logger:            rt.log,
	})
	internalCtx, stopInternal := context.WithCancel(context.WithoutCancel(ctx))
	defer stopInternal()
	internalErr := make(chan error, 1)
	go func() { internalErr <- httpx.Serve(internalCtx, internalSrv, 5*time.Second, rt.log) }()

	stopReadiness := context.AfterFunc(ctx, func() {
		rt.health.SetReady(false)
		rt.log.Info("shutdown signal received; readiness disabled")
	})
	defer stopReadiness()

	var err error
	if public != nil {
		err = httpx.Serve(ctx, public, rt.cfg.HTTP.ShutdownTimeout, rt.log)
	} else {
		<-ctx.Done()
	}
	stopInternal()
	return errors.Join(err, <-internalErr)
}

// publicHandler builds the public route table and middleware pipeline.
//
// Pipeline (outermost first): request ID → OTel span → access log → panic
// recovery → security headers → body limit → CSRF → routes. Authentication,
// organization resolution and rate limiting are added in Gate 1.
func (rt *runtime) publicHandler() (http.Handler, error) {
	csrf, err := httpx.CSRF(rt.log, trustedOrigins(rt.cfg)...)
	if err != nil {
		return nil, err
	}

	apiMux := http.NewServeMux()
	// Unknown /v1 routes answer with a problem document, never the SPA.
	apiMux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteError(w, r, rt.log, apperr.New(apperr.CodeResourceNotFound, "No such API route."))
	})
	routes := api.Handler(&api.Server{}, apiMux, rt.log)

	apiChain := httpx.Chain(routes,
		httpx.SecurityHeaders(rt.cfg.IsProduction(), ""),
		httpx.MaxBytes(rt.cfg.HTTP.MaxBodyBytes),
		csrf,
	)
	spa := httpx.Chain(webui.Handler("/v1/", "/auth/"),
		httpx.SecurityHeaders(rt.cfg.IsProduction(), webui.CSP(nil, nil)),
	)

	root := http.NewServeMux()
	root.Handle("/v1/", apiChain)
	root.Handle("/", spa)

	return httpx.Chain(root,
		httpx.RequestID(false),
		func(next http.Handler) http.Handler {
			return otelhttp.NewHandler(next, "http.server",
				otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
					if r.Pattern != "" {
						return r.Pattern
					}
					return r.Method + " unmatched"
				}))
		},
		httpx.AccessLog(rt.log),
		httpx.Recover(rt.log),
	), nil
}

// trustedOrigins returns origins allowed to send cross-origin unsafe
// requests. In development the Vite dev server proxies to the API, so its
// origin is trusted; in production the SPA is same-origin (ADR-020) and none
// are needed.
func trustedOrigins(cfg config.Config) []string {
	if cfg.IsProduction() {
		return nil
	}
	return []string{cfg.HTTP.PublicOrigin}
}
