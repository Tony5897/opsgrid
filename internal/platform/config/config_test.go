package config_test

import (
	"errors"
	"io/fs"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/config"
)

func env(m map[string]string) config.LookupFunc {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func noFiles(string) ([]byte, error) { return nil, fs.ErrNotExist }

var validAPI = map[string]string{
	"DATABASE_URL": "postgres://opsgrid_app:pw@localhost:5432/opsgrid",
	"REDIS_URL":    "redis://localhost:6379/0",
}

func TestLoadDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := config.LoadFrom(config.RoleAPI, env(validAPI), noFiles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env != config.EnvDevelopment || cfg.HTTP.Addr != ":8080" || cfg.HTTP.InternalAddr != ":9090" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.HTTP.ShutdownTimeout != 30*time.Second || cfg.Log.Level != slog.LevelInfo {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.ServiceName != "opsgrid-api" {
		t.Fatalf("service name = %q", cfg.ServiceName)
	}
}

func TestLoadAggregatesAllErrors(t *testing.T) {
	t.Parallel()
	_, err := config.LoadFrom(config.RoleAPI, env(map[string]string{
		"OPSGRID_ENV":        "staging",
		"LOG_FORMAT":         "xml",
		"SHUTDOWN_TIMEOUT":   "soon",
		"DATABASE_MAX_CONNS": "x",
	}), noFiles)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"OPSGRID_ENV", "LOG_FORMAT", "SHUTDOWN_TIMEOUT", "DATABASE_MAX_CONNS", "DATABASE_URL is required", "REDIS_URL is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q:\n%v", want, err)
		}
	}
}

func TestInvalidURLNeverEchoesValue(t *testing.T) {
	t.Parallel()
	m := map[string]string{"DATABASE_URL": "mysql://user:supersecret@host/db", "REDIS_URL": "redis://x"}
	_, err := config.LoadFrom(config.RoleWorker, env(m), noFiles)
	if err == nil || strings.Contains(err.Error(), "supersecret") {
		t.Fatalf("expected scheme error without secret, got: %v", err)
	}
}

func TestSecretFromFile(t *testing.T) {
	t.Parallel()
	m := map[string]string{"DATABASE_URL_FILE": "/run/secrets/db", "REDIS_URL": "redis://localhost:6379"}
	read := func(p string) ([]byte, error) {
		if p == "/run/secrets/db" {
			return []byte("postgres://app@db/opsgrid\n"), nil
		}
		return nil, errors.New("unexpected path")
	}
	cfg, err := config.LoadFrom(config.RoleRelay, env(m), read)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Database.URL != "postgres://app@db/opsgrid" {
		t.Fatalf("secret not trimmed/loaded: %q", cfg.Database.URL)
	}
}

func TestProductionRequiresHTTPSOrigin(t *testing.T) {
	t.Parallel()
	m := map[string]string{"OPSGRID_ENV": "production", "PUBLIC_ORIGIN": "http://opsgrid.example"}
	for k, v := range validAPI {
		m[k] = v
	}
	if _, err := config.LoadFrom(config.RoleAPI, env(m), noFiles); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected https error, got %v", err)
	}
}

func TestMigrateRoleOnlyNeedsMigratorURL(t *testing.T) {
	t.Parallel()
	m := map[string]string{"MIGRATOR_DATABASE_URL": "postgres://opsgrid_migrator@db/opsgrid"}
	if _, err := config.LoadFrom(config.RoleMigrate, env(m), noFiles); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
