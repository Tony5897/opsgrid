package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// ServerOptions configures NewServer.
type ServerOptions struct {
	Addr              string
	Handler           http.Handler
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	Logger            *slog.Logger
}

// NewServer builds an http.Server with explicit timeouts (no zero-value
// "wait forever" defaults) and a structured error log. Request contexts
// derive from ctx's values but are NOT cancelled when ctx is: shutdown is
// driven by Serve, which lets in-flight requests drain.
func NewServer(ctx context.Context, o ServerOptions) *http.Server {
	srv := &http.Server{
		Addr:              o.Addr,
		Handler:           o.Handler,
		ReadHeaderTimeout: o.ReadHeaderTimeout,
		ReadTimeout:       o.ReadTimeout,
		WriteTimeout:      o.WriteTimeout,
		IdleTimeout:       o.IdleTimeout,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          slog.NewLogLogger(o.Logger.Handler(), slog.LevelWarn),
	}
	base := context.WithoutCancel(ctx)
	srv.BaseContext = func(net.Listener) context.Context { return base }
	return srv
}

// Serve listens on srv.Addr and runs srv until ctx is cancelled, then shuts
// it down gracefully within timeout. It returns nil on a clean shutdown.
func Serve(ctx context.Context, srv *http.Server, timeout time.Duration, log *slog.Logger) error {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", srv.Addr)
	if err != nil {
		return err
	}
	return ServeListener(ctx, srv, ln, timeout, log)
}

// ServeListener is Serve on an existing listener.
func ServeListener(ctx context.Context, srv *http.Server, ln net.Listener, timeout time.Duration, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", slog.String("addr", ln.Addr().String()))
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()
	log.Info("http server shutting down", slog.String("addr", ln.Addr().String()), slog.Duration("timeout", timeout))
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return <-errCh
}
