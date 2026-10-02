package app

import (
	"context"
	"fmt"
	"os"

	"github.com/Tony5897/opsgrid/internal/platform/config"
	"github.com/Tony5897/opsgrid/internal/platform/database"
	"github.com/Tony5897/opsgrid/internal/platform/telemetry"
)

// RunMigrate applies (command "up") or reports ("status") the embedded
// migrations as the schema-owner role. It is a one-shot process that Compose
// runs before the API, worker and relay start.
func RunMigrate(ctx context.Context, command string) error {
	cfg, err := config.Load(config.RoleMigrate)
	if err != nil {
		return err
	}
	log := telemetry.NewLogger(os.Stdout, cfg.Log.Level, cfg.Log.Format)

	m, err := database.NewMigrator(cfg.Database.MigratorURL)
	if err != nil {
		return err
	}
	defer m.Close()

	switch command {
	case "up":
		return m.Up(ctx, log)
	case "status":
		return m.Status(ctx, log)
	default:
		return fmt.Errorf("unknown migrate command %q (want up or status)", command)
	}
}
