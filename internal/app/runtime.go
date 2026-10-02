// Package app is the composition root: the only package that wires
// platform components and domain modules together. cmd/* mains call into it.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Tony5897/opsgrid/internal/platform/buildinfo"
	"github.com/Tony5897/opsgrid/internal/platform/config"
	"github.com/Tony5897/opsgrid/internal/platform/database"
	"github.com/Tony5897/opsgrid/internal/platform/health"
	"github.com/Tony5897/opsgrid/internal/platform/redisx"
	"github.com/Tony5897/opsgrid/internal/platform/telemetry"
)

// runtime holds the infrastructure shared by long-running processes.
type runtime struct {
	cfg    config.Config
	log    *slog.Logger
	db     *database.DB
	redis  *redis.Client
	health *health.Health

	closers []func(context.Context) error
}

// newRuntime loads configuration, then starts telemetry, PostgreSQL and
// Redis. PostgreSQL is required at boot (fail fast); Redis is opened lazily
// and only health-checked, because the API must keep serving synchronous
// operations during a Redis outage (failure matrix).
func newRuntime(ctx context.Context, role config.Role) (*runtime, error) {
	cfg, err := config.Load(role)
	if err != nil {
		return nil, err
	}
	log := telemetry.NewLogger(os.Stdout, cfg.Log.Level, cfg.Log.Format).
		With(slog.String("service", cfg.ServiceName), slog.String("version", buildinfo.Version))
	slog.SetDefault(log)

	rt := &runtime{cfg: cfg, log: log}

	shutdownTel, err := telemetry.Setup(ctx, telemetry.Options{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: buildinfo.Version + "+" + buildinfo.Short(),
		Environment:    cfg.Env,
		OTLPEndpoint:   cfg.Telemetry.OTLPEndpoint,
		Insecure:       cfg.Telemetry.OTLPInsecure,
		SampleRatio:    cfg.Telemetry.SampleRatio,
	})
	if err != nil {
		return nil, err
	}
	rt.closers = append(rt.closers, shutdownTel)

	rt.db, err = database.Open(ctx, database.Options{
		URL:               cfg.Database.URL,
		MaxConns:          cfg.Database.MaxConns,
		MinConns:          cfg.Database.MinConns,
		MaxConnLifetime:   cfg.Database.MaxConnLifetime,
		HealthCheckPeriod: cfg.Database.HealthCheckPeriod,
		ApplicationName:   cfg.ServiceName,
	}, log)
	if err != nil {
		rt.close(ctx)
		return nil, err
	}
	rt.closers = append(rt.closers, func(context.Context) error { rt.db.Close(); return nil })

	rt.redis, err = redisx.Open(cfg.Redis.URL)
	if err != nil {
		rt.close(ctx)
		return nil, err
	}
	rt.closers = append(rt.closers, func(context.Context) error { return rt.redis.Close() })

	log.Info("process starting",
		slog.String("role", string(role)),
		slog.String("env", cfg.Env),
		slog.String("commit", buildinfo.Short()),
		slog.Bool("otlp_export", cfg.Telemetry.OTLPEndpoint != ""))
	return rt, nil
}

// baseChecks are the readiness checks every process shares. redisCritical
// is false for the API (degradable) and true for background processes,
// which cannot do any work without Redis.
func (rt *runtime) baseChecks(redisCritical bool) []health.Check {
	return []health.Check{
		{Name: "postgres", Critical: true, Fn: rt.db.Ping},
		{Name: "schema", Critical: true, Fn: rt.db.CheckMigrations},
		{Name: "redis", Critical: redisCritical, Fn: redisx.Ping(rt.redis)},
	}
}

// close releases resources in reverse order of acquisition.
func (rt *runtime) close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	var errs []error
	for i := len(rt.closers) - 1; i >= 0; i-- {
		if err := rt.closers[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
