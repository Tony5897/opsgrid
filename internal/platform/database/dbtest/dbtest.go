//go:build integration

// Package dbtest provides real PostgreSQL databases for integration tests.
//
// One container is started per test binary. It is bootstrapped with the same
// db/bootstrap/roles.sql used in every environment, a template database is
// migrated once, and each test receives a fresh clone (CREATE DATABASE ...
// TEMPLATE), which takes milliseconds and gives full isolation.
package dbtest

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Tony5897/opsgrid/db"
	"github.com/Tony5897/opsgrid/internal/platform/database"
)

// Image is the PostgreSQL image used for integration tests. Keep in sync with
// compose.yaml (Renovate updates both).
const Image = "postgres:18.6"

const (
	templateDB = "opsgrid_tpl"
	// RolePassword is set on every role inside the throwaway container only.
	RolePassword = "test-only-password"
)

// Env is a running PostgreSQL container with a migrated template database.
type Env struct {
	container *tcpostgres.PostgresContainer
	superURL  *url.URL
	counter   atomic.Int64
}

var env *Env

// Main starts the shared container, runs the tests and tears it down. Use it
// from TestMain: func TestMain(m *testing.M) { dbtest.Main(m) }.
func Main(m *testing.M) {
	ctx := context.Background()
	var err error
	env, err = start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbtest: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = testcontainers.TerminateContainer(env.container)
	os.Exit(code)
}

func start(ctx context.Context) (*Env, error) {
	c, err := tcpostgres.Run(ctx, Image,
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres: %w", err)
	}
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	e := &Env{container: c, superURL: u}
	if err := e.bootstrap(ctx); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Env) urlFor(user, password, dbname string) string {
	u := *e.superURL
	u.User = url.UserPassword(user, password)
	u.Path = "/" + dbname
	return u.String()
}

func (e *Env) superConn(ctx context.Context, dbname string) (*pgx.Conn, error) {
	u := *e.superURL
	u.Path = "/" + dbname
	return pgx.Connect(ctx, u.String())
}

func (e *Env) bootstrap(ctx context.Context) error {
	conn, err := e.superConn(ctx, "postgres")
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	roles, err := db.Bootstrap("roles.sql")
	if err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, roles); err != nil {
		return fmt.Errorf("roles.sql: %w", err)
	}
	for _, r := range Roles {
		if _, err := conn.Exec(ctx, fmt.Sprintf("ALTER ROLE %s PASSWORD '%s'", r, RolePassword)); err != nil {
			return err
		}
	}
	if err := createDB(ctx, conn, templateDB, ""); err != nil {
		return err
	}

	m, err := database.NewMigrator(e.urlFor("opsgrid_migrator", RolePassword, templateDB))
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(ctx, slog.New(slog.DiscardHandler)); err != nil {
		return err
	}
	// A template must have no connections when cloned.
	_, err = conn.Exec(ctx, fmt.Sprintf("ALTER DATABASE %s WITH ALLOW_CONNECTIONS false IS_TEMPLATE true", templateDB))
	return err
}

func createDB(ctx context.Context, conn *pgx.Conn, name, template string) error {
	stmt := fmt.Sprintf("CREATE DATABASE %s OWNER opsgrid_migrator", name)
	if template != "" {
		stmt += " TEMPLATE " + template
	}
	if _, err := conn.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("create database %s: %w", name, err)
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf("REVOKE ALL ON DATABASE %s FROM PUBLIC", name)); err != nil {
		return err
	}
	_, err := conn.Exec(ctx, fmt.Sprintf(
		"GRANT CONNECT ON DATABASE %s TO opsgrid_migrator, opsgrid_app, opsgrid_worker, opsgrid_relay, opsgrid_reporting_ro", name))
	return err
}

// Roles bootstrapped in every environment.
var Roles = []string{"opsgrid_migrator", "opsgrid_app", "opsgrid_worker", "opsgrid_relay", "opsgrid_reporting_ro"}

// Database is a fresh, migrated database for one test.
type Database struct {
	Name string
	env  *Env
}

// URL returns a connection URL for role (e.g. "opsgrid_app").
func (d *Database) URL(role string) string {
	return d.env.urlFor(role, RolePassword, d.Name)
}

// SuperURL returns a superuser connection URL. Only for test assertions that
// must inspect catalogs or bypass RLS deliberately.
func (d *Database) SuperURL() string {
	u := *d.env.superURL
	u.Path = "/" + d.Name
	return u.String()
}

// New clones the migrated template into a new database dropped at cleanup.
func New(t testing.TB) *Database {
	t.Helper()
	if env == nil {
		t.Fatal("dbtest: call dbtest.Main from TestMain")
	}
	ctx := context.Background()
	name := fmt.Sprintf("t_%s_%d", sanitize(t.Name()), env.counter.Add(1))
	conn, err := env.superConn(ctx, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if err := createDB(ctx, conn, name, templateDB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, err := env.superConn(context.Background(), "postgres")
		if err != nil {
			return
		}
		defer c.Close(context.Background())
		_, _ = c.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name))
	})
	return &Database{Name: name, env: env}
}

// Open returns a database.DB connected as role.
func (d *Database) Open(t testing.TB, role string, maxConns int32) *database.DB {
	t.Helper()
	dbh, err := database.Open(context.Background(), database.Options{
		URL: d.URL(role), MaxConns: maxConns, MinConns: 0, ApplicationName: "dbtest",
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(dbh.Close)
	return dbh
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}
