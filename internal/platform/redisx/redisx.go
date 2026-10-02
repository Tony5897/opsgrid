// Package redisx creates instrumented Redis clients.
package redisx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// Open parses url, applies conservative timeouts and instruments the client
// with OpenTelemetry tracing and metrics. Connectivity is NOT required at
// boot: Redis is a degradable dependency for the API (failure matrix), so
// callers decide whether a failed Ping is fatal.
func Open(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, errors.New("redis: invalid URL") // never echo credentials
	}
	opt.DialTimeout = 2 * time.Second
	opt.ReadTimeout = 500 * time.Millisecond // blocking commands (XREADGROUP BLOCK) extend this per call
	opt.WriteTimeout = 500 * time.Millisecond
	opt.PoolTimeout = time.Second
	opt.ContextTimeoutEnabled = true

	c := redis.NewClient(opt)
	if err := redisotel.InstrumentTracing(c); err != nil {
		return nil, fmt.Errorf("redis tracing: %w", err)
	}
	if err := redisotel.InstrumentMetrics(c); err != nil {
		return nil, fmt.Errorf("redis metrics: %w", err)
	}
	return c, nil
}

// Ping returns a health-check function for c.
func Ping(c *redis.Client) func(context.Context) error {
	return func(ctx context.Context) error { return c.Ping(ctx).Err() }
}
