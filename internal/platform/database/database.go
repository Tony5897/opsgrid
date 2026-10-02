// Package database owns every PostgreSQL connection and transaction in
// OpsGrid. It is the only package allowed to begin a transaction (enforced by
// lint), which guarantees that tenant-owned tables are only ever touched with
// a transaction-local tenant context (ADR-004).
package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/Tony5897/opsgrid/db"
)

const tracerName = "github.com/Tony5897/opsgrid/internal/platform/database"

// Options configures Open.
type Options struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	HealthCheckPeriod time.Duration
	// ApplicationName appears in pg_stat_activity.
	ApplicationName string
}

// DB wraps a pgx pool. Callers never receive the pool for tenant work; they
// use InTenantTx / InUserTx.
type DB struct {
	pool   *pgxpool.Pool
	log    *slog.Logger
	tracer trace.Tracer

	// retry policy for serialization failures and deadlocks
	maxAttempts int
	baseBackoff time.Duration
}

// Open creates the pool and verifies connectivity.
func Open(ctx context.Context, o Options, log *slog.Logger) (*DB, error) {
	pc, err := pgxpool.ParseConfig(o.URL)
	if err != nil {
		return nil, errors.New("database: invalid connection URL") // never echo the URL
	}
	if o.MaxConns > 0 {
		pc.MaxConns = o.MaxConns
	}
	if o.MinConns > 0 {
		pc.MinConns = o.MinConns
	}
	if o.MaxConnLifetime > 0 {
		pc.MaxConnLifetime = o.MaxConnLifetime
		pc.MaxConnLifetimeJitter = o.MaxConnLifetime / 10
	}
	if o.HealthCheckPeriod > 0 {
		pc.HealthCheckPeriod = o.HealthCheckPeriod
	}
	if o.ApplicationName != "" {
		pc.ConnConfig.RuntimeParams["application_name"] = o.ApplicationName
	}
	pc.ConnConfig.Tracer = otelpgx.NewTracer() // query parameters are excluded by default (may contain PII)

	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("database: create pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}
	if err := otelpgx.RecordStats(pool); err != nil {
		log.Warn("database: pool metrics unavailable", slog.Any("error", err))
	}
	return &DB{
		pool:        pool,
		log:         log,
		tracer:      otel.Tracer(tracerName),
		maxAttempts: 3,
		baseBackoff: 20 * time.Millisecond,
	}, nil
}

// Close releases all connections.
func (d *DB) Close() { d.pool.Close() }

// Ping checks connectivity (used by readiness).
func (d *DB) Ping(ctx context.Context) error { return d.pool.Ping(ctx) }

// Scope identifies the tenant and actor for a transaction.
type Scope struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID // may be uuid.Nil for system actors (workers)
}

// ErrMissingTenant is returned when InTenantTx is called without an
// organization. It is a programming error and fails closed.
var ErrMissingTenant = errors.New("database: tenant transaction requires an organization id")

// TxFunc is the body of a transaction. It may be executed more than once if
// the transaction is retried after a serialization failure or deadlock, so it
// must not have side effects outside the transaction.
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// InTenantTx runs fn in a READ COMMITTED transaction whose tenant context
// (app.organization_id, app.user_id) is set with set_config(..., true): it is
// scoped to this transaction and can never leak to the next borrower of the
// pooled connection.
func (d *DB) InTenantTx(ctx context.Context, s Scope, fn TxFunc) error {
	if s.OrganizationID == uuid.Nil {
		return ErrMissingTenant
	}
	return d.run(ctx, "db.tenant_tx", func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`SELECT set_config('app.organization_id', $1, true), set_config('app.user_id', $2, true)`,
			s.OrganizationID.String(), nilableUUID(s.UserID),
		); err != nil {
			return fmt.Errorf("set tenant context: %w", err)
		}
		return fn(ctx, tx)
	}, attribute.String("opsgrid.organization_id", s.OrganizationID.String()))
}

// InUserTx runs fn with only app.user_id set. It is for pre-tenant lookups
// (sessions, the caller's own memberships) whose RLS policies key on the
// acting user rather than an organization.
func (d *DB) InUserTx(ctx context.Context, userID uuid.UUID, fn TxFunc) error {
	if userID == uuid.Nil {
		return errors.New("database: user transaction requires a user id")
	}
	return d.run(ctx, "db.user_tx", func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, userID.String()); err != nil {
			return fmt.Errorf("set user context: %w", err)
		}
		return fn(ctx, tx)
	})
}

// InSystemTx runs fn without any tenant or user context. Under RLS it can only
// see non-tenant tables; it exists for identity bootstrap (user upsert at
// login, session lookup by hash) and maintenance jobs.
func (d *DB) InSystemTx(ctx context.Context, fn TxFunc) error {
	return d.run(ctx, "db.system_tx", fn)
}

func (d *DB) run(ctx context.Context, spanName string, fn TxFunc, attrs ...attribute.KeyValue) error {
	ctx, span := d.tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))
	defer span.End()

	var err error
	for attempt := 1; attempt <= d.maxAttempts; attempt++ {
		span.SetAttributes(attribute.Int("db.tx.attempt", attempt))
		err = pgx.BeginTxFunc(ctx, d.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
			return fn(ctx, tx)
		})
		if err == nil {
			return nil
		}
		if !IsRetryable(err) || attempt == d.maxAttempts {
			break
		}
		backoff := d.baseBackoff * time.Duration(1<<(attempt-1))
		sleep := backoff/2 + rand.N(backoff/2+1) //nolint:gosec // G404: retry jitter, not security-sensitive
		d.log.WarnContext(ctx, "database: retrying transaction",
			slog.Int("attempt", attempt), slog.Duration("backoff", sleep), slog.String("sqlstate", SQLState(err)))
		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			err = errors.Join(err, ctx.Err())
			span.RecordError(err)
			span.SetStatus(codes.Error, "context done during retry")
			return err
		}
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, "transaction failed")
	return err
}

// MigrationVersion returns the highest applied goose migration version.
func (d *DB) MigrationVersion(ctx context.Context) (int64, error) {
	var v int64
	err := d.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(version_id), 0) FROM public.goose_db_version WHERE is_applied`).Scan(&v)
	return v, err
}

// CheckMigrations reports an error unless the database schema is exactly at
// the version embedded in this binary.
func (d *DB) CheckMigrations(ctx context.Context) error {
	want, err := db.LatestVersion()
	if err != nil {
		return err
	}
	got, err := d.MigrationVersion(ctx)
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if got != want {
		return fmt.Errorf("schema version %d, binary expects %d", got, want)
	}
	return nil
}

func nilableUUID(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}
