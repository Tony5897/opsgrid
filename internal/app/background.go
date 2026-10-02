package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/config"
	"github.com/Tony5897/opsgrid/internal/platform/health"
)

// RunWorker runs the stream-consumer process. Handlers (notifications,
// exports, reports, maintenance) are registered in Gate 6; until then the
// process boots, reports health and shuts down cleanly, so the Compose
// topology and deployment pipeline are exercised from Gate 0.
func RunWorker(ctx context.Context) error {
	return runBackground(ctx, config.RoleWorker, func(ctx context.Context, rt *runtime) error {
		rt.log.Info("worker ready; no consumer groups registered yet (Gate 6)")
		<-ctx.Done()
		return nil
	})
}

// RunRelay runs the transactional-outbox relay (ADR-007). The publish loop
// arrives in Gate 6 together with the outbox table.
func RunRelay(ctx context.Context) error {
	return runBackground(ctx, config.RoleRelay, func(ctx context.Context, rt *runtime) error {
		rt.log.Info("relay ready; outbox publishing arrives in Gate 6")
		<-ctx.Done()
		return nil
	})
}

// runBackground wires the shared runtime and internal health server around a
// long-running loop. The loop must return promptly after ctx is cancelled;
// it is given the configured shutdown timeout to finish in-flight work.
func runBackground(ctx context.Context, role config.Role, loop func(context.Context, *runtime) error) (err error) {
	rt, err := newRuntime(ctx, role)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rt.close(ctx)) }()
	rt.health = health.New(2*time.Second, rt.baseChecks(true)...)

	loopErr := make(chan error, 1)
	go func() { loopErr <- loop(ctx, rt) }()

	serveErr := make(chan error, 1)
	serveCtx, stopServe := context.WithCancel(ctx)
	defer stopServe()
	go func() { serveErr <- rt.serveWithInternal(serveCtx, nil) }()

	select {
	case err = <-loopErr:
		// The loop exited on its own (fatal error or ctx done).
		stopServe()
		return errors.Join(err, <-serveErr)
	case <-ctx.Done():
	}

	rt.log.Info("draining background work", slog.Duration("timeout", rt.cfg.HTTP.ShutdownTimeout))
	select {
	case err = <-loopErr:
	case <-time.After(rt.cfg.HTTP.ShutdownTimeout):
		err = errors.New("background loop did not stop within the shutdown timeout")
	}
	return errors.Join(err, <-serveErr)
}
