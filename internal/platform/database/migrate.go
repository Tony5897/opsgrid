package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/Tony5897/opsgrid/db"
)

// Migrator applies the embedded migrations. It must connect as
// opsgrid_migrator (the schema owner), never as a runtime role.
type Migrator struct {
	provider *goose.Provider
	sqlDB    *sql.DB
}

// NewMigrator opens a dedicated connection for migrations.
func NewMigrator(migratorURL string) (*Migrator, error) {
	cfg, err := stdlibConfig(migratorURL)
	if err != nil {
		return nil, err
	}
	sqlDB := stdlib.OpenDB(*cfg)
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations())
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrator: %w", err)
	}
	return &Migrator{provider: p, sqlDB: sqlDB}, nil
}

// Up applies all pending migrations.
func (m *Migrator) Up(ctx context.Context, log *slog.Logger) error {
	results, err := m.provider.Up(ctx)
	for _, r := range results {
		log.Info("migration applied",
			slog.Int64("version", r.Source.Version),
			slog.String("file", r.Source.Path),
			slog.Duration("duration", r.Duration))
	}
	if err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	v, err := m.provider.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	log.Info("database schema up to date", slog.Int64("version", v))
	return nil
}

// Status logs each migration's state.
func (m *Migrator) Status(ctx context.Context, log *slog.Logger) error {
	statuses, err := m.provider.Status(ctx)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		log.Info("migration", slog.Int64("version", s.Source.Version),
			slog.String("file", s.Source.Path), slog.String("state", string(s.State)))
	}
	return nil
}

// Close closes the migrator's connection.
func (m *Migrator) Close() error {
	return errors.Join(m.provider.Close(), m.sqlDB.Close())
}
