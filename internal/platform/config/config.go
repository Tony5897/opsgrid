// Package config loads typed, validated process configuration from the
// environment. Every process fails fast at boot on invalid configuration.
//
// Secrets may be supplied either directly (FOO) or as a path to a file
// containing the value (FOO_FILE), which suits Docker/Kubernetes secrets.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment names.
const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"
)

// Config is the union of settings used by OpsGrid processes. Each process
// reads only what it needs; Load validates fields marked required for the
// requested Role.
type Config struct {
	Env         string
	ServiceName string

	HTTP      HTTP
	Database  Database
	Redis     Redis
	Telemetry Telemetry
	Log       Log
}

type HTTP struct {
	// Addr serves public traffic (API, WebSocket, SPA).
	Addr string
	// InternalAddr serves /livez, /readyz and /metrics; never exposed publicly.
	InternalAddr      string
	PublicOrigin      string // e.g. http://localhost:5173 in dev; used for CSRF and WS origin checks
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxBodyBytes      int64
}

type Database struct {
	URL               string // runtime role (opsgrid_app / opsgrid_worker / opsgrid_relay)
	MigratorURL       string // only used by cmd/migrate
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	HealthCheckPeriod time.Duration
}

type Redis struct {
	URL string
}

type Telemetry struct {
	// OTLPEndpoint is the gRPC endpoint of the OpenTelemetry Collector
	// (host:port). Empty disables export; propagation still works.
	OTLPEndpoint string
	// OTLPInsecure disables TLS to the collector (true on the local Compose
	// network; must be false in production).
	OTLPInsecure bool
	SampleRatio  float64
}

type Log struct {
	Level  slog.Level
	Format string // json | text
}

// Role selects which settings are required.
type Role string

const (
	RoleAPI     Role = "api"
	RoleWorker  Role = "worker"
	RoleRelay   Role = "relay"
	RoleMigrate Role = "migrate"
)

// Load reads configuration for role from the process environment.
func Load(role Role) (Config, error) {
	return LoadFrom(role, os.LookupEnv, os.ReadFile)
}

// LookupFunc matches os.LookupEnv; ReadFileFunc matches os.ReadFile.
type (
	LookupFunc   func(string) (string, bool)
	ReadFileFunc func(string) ([]byte, error)
)

// LoadFrom is Load with injectable sources, for tests.
func LoadFrom(role Role, lookup LookupFunc, readFile ReadFileFunc) (Config, error) {
	r := reader{lookup: lookup, readFile: readFile}

	cfg := Config{
		Env:         r.str("OPSGRID_ENV", EnvDevelopment),
		ServiceName: r.str("OTEL_SERVICE_NAME", "opsgrid-"+string(role)),
		HTTP: HTTP{
			Addr:              r.str("HTTP_ADDR", ":8080"),
			InternalAddr:      r.str("HTTP_INTERNAL_ADDR", ":9090"),
			PublicOrigin:      r.str("PUBLIC_ORIGIN", "http://localhost:5173"),
			ReadHeaderTimeout: r.duration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       r.duration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      r.duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:       r.duration("HTTP_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout:   r.duration("SHUTDOWN_TIMEOUT", 30*time.Second),
			MaxBodyBytes:      r.int64("HTTP_MAX_BODY_BYTES", 1<<20),
		},
		Database: Database{
			URL:               r.secret("DATABASE_URL"),
			MigratorURL:       r.secret("MIGRATOR_DATABASE_URL"),
			MaxConns:          r.int32("DATABASE_MAX_CONNS", 20),
			MinConns:          r.int32("DATABASE_MIN_CONNS", 2),
			MaxConnLifetime:   r.duration("DATABASE_MAX_CONN_LIFETIME", time.Hour),
			HealthCheckPeriod: r.duration("DATABASE_HEALTH_CHECK_PERIOD", 30*time.Second),
		},
		Redis: Redis{URL: r.secret("REDIS_URL")},
		Telemetry: Telemetry{
			OTLPEndpoint: r.str("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			OTLPInsecure: r.boolean("OTEL_EXPORTER_OTLP_INSECURE", true),
			SampleRatio:  r.float("OTEL_TRACES_SAMPLER_RATIO", 1.0),
		},
		Log: Log{
			Level:  r.level("LOG_LEVEL", slog.LevelInfo),
			Format: r.str("LOG_FORMAT", "json"),
		},
	}

	cfg.validate(role, &r)
	if len(r.errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(r.errs...))
	}
	return cfg, nil
}

func (c *Config) validate(role Role, r *reader) {
	switch c.Env {
	case EnvDevelopment, EnvTest, EnvProduction:
	default:
		r.fail("OPSGRID_ENV must be one of development, test, production; got %q", c.Env)
	}
	switch c.Log.Format {
	case "json", "text":
	default:
		r.fail("LOG_FORMAT must be json or text; got %q", c.Log.Format)
	}
	if c.Env == EnvProduction && c.Telemetry.OTLPEndpoint != "" && c.Telemetry.OTLPInsecure {
		r.fail("OTEL_EXPORTER_OTLP_INSECURE must be false in production")
	}
	if c.Telemetry.SampleRatio < 0 || c.Telemetry.SampleRatio > 1 {
		r.fail("OTEL_TRACES_SAMPLER_RATIO must be within [0,1]; got %v", c.Telemetry.SampleRatio)
	}

	switch role {
	case RoleMigrate:
		requireURL(r, "MIGRATOR_DATABASE_URL", c.Database.MigratorURL, "postgres", "postgresql")
		return
	case RoleAPI, RoleWorker, RoleRelay:
		requireURL(r, "DATABASE_URL", c.Database.URL, "postgres", "postgresql")
		requireURL(r, "REDIS_URL", c.Redis.URL, "redis", "rediss")
	default:
		r.fail("unknown role %q", role)
	}

	if role == RoleAPI {
		requireURL(r, "PUBLIC_ORIGIN", c.HTTP.PublicOrigin, "http", "https")
		if c.Env == EnvProduction && strings.HasPrefix(c.HTTP.PublicOrigin, "http://") {
			r.fail("PUBLIC_ORIGIN must use https in production")
		}
	}
	if c.Database.MinConns < 0 || c.Database.MaxConns < 1 || c.Database.MinConns > c.Database.MaxConns {
		r.fail("DATABASE_MIN_CONNS/MAX_CONNS invalid: min=%d max=%d", c.Database.MinConns, c.Database.MaxConns)
	}
	if c.HTTP.MaxBodyBytes <= 0 {
		r.fail("HTTP_MAX_BODY_BYTES must be positive")
	}
}

func requireURL(r *reader, name, value string, schemes ...string) {
	if value == "" {
		r.fail("%s is required", name)
		return
	}
	u, err := url.Parse(value)
	if err != nil {
		// Never echo the value: it may contain credentials.
		r.fail("%s is not a valid URL", name)
		return
	}
	for _, s := range schemes {
		if u.Scheme == s {
			return
		}
	}
	r.fail("%s must use scheme %s", name, strings.Join(schemes, " or "))
}

// IsProduction reports whether the process runs in production.
func (c Config) IsProduction() bool { return c.Env == EnvProduction }

type reader struct {
	lookup   LookupFunc
	readFile ReadFileFunc
	errs     []error
}

func (r *reader) fail(format string, args ...any) {
	r.errs = append(r.errs, fmt.Errorf(format, args...))
}

func (r *reader) raw(name string) (string, bool) {
	v, ok := r.lookup(name)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(v), true
}

func (r *reader) str(name, def string) string {
	if v, ok := r.raw(name); ok && v != "" {
		return v
	}
	return def
}

// secret reads NAME, or the contents of the file named by NAME_FILE.
func (r *reader) secret(name string) string {
	if v, ok := r.raw(name); ok && v != "" {
		return v
	}
	path, ok := r.raw(name + "_FILE")
	if !ok || path == "" {
		return ""
	}
	b, err := r.readFile(path)
	if err != nil {
		r.fail("%s_FILE: cannot read secret file: %v", name, err)
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (r *reader) duration(name string, def time.Duration) time.Duration {
	v, ok := r.raw(name)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < 0 {
		r.fail("%s must be a non-negative duration (e.g. 5s); got %q", name, v)
		return def
	}
	return d
}

func (r *reader) int64(name string, def int64) int64 {
	v, ok := r.raw(name)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		r.fail("%s must be an integer; got %q", name, v)
		return def
	}
	return n
}

func (r *reader) int32(name string, def int32) int32 {
	v, ok := r.raw(name)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		r.fail("%s must be a 32-bit integer; got %q", name, v)
		return def
	}
	return int32(n)
}

func (r *reader) float(name string, def float64) float64 {
	v, ok := r.raw(name)
	if !ok || v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		r.fail("%s must be a number; got %q", name, v)
		return def
	}
	return f
}

func (r *reader) boolean(name string, def bool) bool {
	v, ok := r.raw(name)
	if !ok || v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		r.fail("%s must be true or false; got %q", name, v)
		return def
	}
	return b
}

func (r *reader) level(name string, def slog.Level) slog.Level {
	v, ok := r.raw(name)
	if !ok || v == "" {
		return def
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(v)); err != nil {
		r.fail("%s must be debug, info, warn or error; got %q", name, v)
		return def
	}
	return l
}
