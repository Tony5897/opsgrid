package app

import (
	"context"
	"os"

	"github.com/Tony5897/opsgrid/internal/platform/config"
	"github.com/Tony5897/opsgrid/internal/platform/database"
	"github.com/Tony5897/opsgrid/internal/platform/seed"
	"github.com/Tony5897/opsgrid/internal/platform/telemetry"
)

// RunSeed upserts demo organizations, users and memberships as opsgrid_app.
func RunSeed(ctx context.Context) error {
	cfg, err := config.Load(config.RoleSeed)
	if err != nil {
		return err
	}
	log := telemetry.NewLogger(os.Stdout, cfg.Log.Level, cfg.Log.Format)

	db, err := database.Open(ctx, database.Options{
		URL:               cfg.Database.URL,
		MaxConns:          4,
		MinConns:          1,
		MaxConnLifetime:   cfg.Database.MaxConnLifetime,
		HealthCheckPeriod: cfg.Database.HealthCheckPeriod,
		ApplicationName:   "opsgrid-seed",
	}, log)
	if err != nil {
		return err
	}
	defer db.Close()

	return seed.Run(ctx, db, log)
}
