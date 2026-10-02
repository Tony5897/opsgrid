//go:build integration

package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Tony5897/opsgrid/internal/platform/database"
	"github.com/Tony5897/opsgrid/internal/platform/database/dbtest"
)

func TestMain(m *testing.M) { dbtest.Main(m) }

// The tenant context must be transaction-local: after the transaction ends,
// the next borrower of the same pooled connection sees no tenant. The pool
// is limited to one connection to force reuse. This also proves the NULLIF
// fix: the raw setting reads back as ” (which ::uuid would reject), yet
// app.current_org_id() is NULL.
func TestTenantContextIsTransactionLocal(t *testing.T) {
	t.Parallel()
	d := dbtest.New(t).Open(t, "opsgrid_app", 1)
	ctx := context.Background()
	org, user := uuid.New(), uuid.New()

	err := d.InTenantTx(ctx, database.Scope{OrganizationID: org, UserID: user}, func(ctx context.Context, tx pgx.Tx) error {
		var gotOrg, gotUser uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT app.current_org_id(), app.current_user_id()`).Scan(&gotOrg, &gotUser); err != nil {
			return err
		}
		if gotOrg != org || gotUser != user {
			t.Errorf("context inside tx = (%s,%s), want (%s,%s)", gotOrg, gotUser, org, user)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = d.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var raw string
		var orgID *uuid.UUID
		if err := tx.QueryRow(ctx,
			`SELECT current_setting('app.organization_id', true), app.current_org_id()`).Scan(&raw, &orgID); err != nil {
			return err
		}
		if raw != "" {
			t.Errorf("raw setting leaked across transactions: %q", raw)
		}
		if orgID != nil {
			t.Errorf("current_org_id() = %s after tx end, want NULL", orgID)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInTenantTxRequiresOrganization(t *testing.T) {
	t.Parallel()
	d := dbtest.New(t).Open(t, "opsgrid_app", 1)
	err := d.InTenantTx(context.Background(), database.Scope{}, func(context.Context, pgx.Tx) error {
		t.Fatal("fn must not run without a tenant")
		return nil
	})
	if !errors.Is(err, database.ErrMissingTenant) {
		t.Fatalf("err = %v, want ErrMissingTenant", err)
	}
}

// The policy template every tenant table uses (ADR-004) is exercised here on
// a scratch table, so the pattern is proven before the first real table.
func TestRLSPolicyTemplateFailsClosed(t *testing.T) {
	t.Parallel()
	tdb := dbtest.New(t)
	ctx := context.Background()

	owner, err := pgx.Connect(ctx, tdb.URL("opsgrid_migrator"))
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close(ctx)
	orgA, orgB := uuid.New(), uuid.New()
	_, err = owner.Exec(ctx, `
		CREATE TABLE app.rls_probe (
			id uuid PRIMARY KEY DEFAULT uuidv7(),
			organization_id uuid NOT NULL,
			label text NOT NULL
		);
		ALTER TABLE app.rls_probe ENABLE ROW LEVEL SECURITY;
		ALTER TABLE app.rls_probe FORCE ROW LEVEL SECURITY;
		CREATE POLICY tenant_isolation ON app.rls_probe TO opsgrid_app, opsgrid_worker
			USING (organization_id = (SELECT app.current_org_id()))
			WITH CHECK (organization_id = (SELECT app.current_org_id()));
	`)
	if err != nil {
		t.Fatal(err)
	}
	// FORCE ROW LEVEL SECURITY applies policies to the table owner too. The
	// owner has no applicable policy, so even with a tenant context it is
	// denied by default: ownership is not a back door.
	tx, err := owner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.organization_id', $1, true)`, orgA.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app.rls_probe (organization_id, label) VALUES ($1, 'row')`, orgA); database.SQLState(err) != database.SQLStateInsufficientPriv {
		t.Fatalf("owner insert under FORCE RLS: err = %v, want 42501", err)
	}
	_ = tx.Rollback(ctx)

	app := tdb.Open(t, "opsgrid_app", 2)
	insert := func(scopeOrg, rowOrg uuid.UUID) error {
		return app.InTenantTx(ctx, database.Scope{OrganizationID: scopeOrg}, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO app.rls_probe (organization_id, label) VALUES ($1, 'row')`, rowOrg)
			return err
		})
	}
	count := func(scope *uuid.UUID) int {
		var n int
		run := func(ctx context.Context, tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM app.rls_probe`).Scan(&n)
		}
		var err error
		if scope == nil {
			err = app.InSystemTx(ctx, run)
		} else {
			err = app.InTenantTx(ctx, database.Scope{OrganizationID: *scope}, run)
		}
		if err != nil {
			t.Fatal(err)
		}
		return n
	}

	if err := insert(orgA, orgA); err != nil {
		t.Fatalf("same-tenant insert failed: %v", err)
	}
	if err := insert(orgB, orgB); err != nil {
		t.Fatalf("same-tenant insert failed: %v", err)
	}
	// WITH CHECK: tenant A cannot write a row labelled as tenant B.
	err = insert(orgA, orgB)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != database.SQLStateInsufficientPriv {
		t.Fatalf("cross-tenant insert: err = %v, want SQLSTATE 42501", err)
	}

	if n := count(&orgA); n != 1 {
		t.Errorf("tenant A sees %d rows, want 1", n)
	}
	if n := count(&orgB); n != 1 {
		t.Errorf("tenant B sees %d rows, want 1", n)
	}
	if n := count(nil); n != 0 {
		t.Errorf("no tenant context sees %d rows, want 0 (fail closed)", n)
	}
}

func TestRuntimeRolesAreHardened(t *testing.T) {
	t.Parallel()
	tdb := dbtest.New(t)
	ctx := context.Background()
	su, err := pgx.Connect(ctx, tdb.SuperURL())
	if err != nil {
		t.Fatal(err)
	}
	defer su.Close(ctx)

	rows, err := su.Query(ctx, `
		SELECT rolname, rolsuper, rolbypassrls, rolcreaterole, rolcreatedb
		FROM pg_roles WHERE rolname LIKE 'opsgrid\_%'`)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for rows.Next() {
		var name string
		var super, bypass, createRole, createDB bool
		if err := rows.Scan(&name, &super, &bypass, &createRole, &createDB); err != nil {
			t.Fatal(err)
		}
		seen++
		if super || bypass || createRole || createDB {
			t.Errorf("%s has elevated attributes: super=%v bypassrls=%v createrole=%v createdb=%v",
				name, super, bypass, createRole, createDB)
		}
	}
	if seen != len(dbtest.Roles) {
		t.Fatalf("found %d opsgrid roles, want %d", seen, len(dbtest.Roles))
	}

	// The runtime role cannot create objects (no DDL at runtime).
	app := tdb.Open(t, "opsgrid_app", 1)
	err = app.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `CREATE TABLE app.should_fail (id int)`)
		return err
	})
	if database.SQLState(err) != database.SQLStateInsufficientPriv {
		t.Fatalf("runtime DDL: err = %v, want 42501", err)
	}

	// Guard-rail settings are applied to the runtime role.
	err = app.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var st string
		if err := tx.QueryRow(ctx, `SHOW statement_timeout`).Scan(&st); err != nil {
			return err
		}
		if st != "5s" {
			t.Errorf("statement_timeout = %s, want 5s", st)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRetriesSerializationFailures(t *testing.T) {
	t.Parallel()
	d := dbtest.New(t).Open(t, "opsgrid_app", 1)
	attempts := 0
	err := d.InSystemTx(context.Background(), func(ctx context.Context, tx pgx.Tx) error {
		attempts++
		if attempts == 1 {
			return &pgconn.PgError{Code: database.SQLStateSerializationFailure}
		}
		return nil
	})
	if err != nil || attempts != 2 {
		t.Fatalf("err=%v attempts=%d, want nil/2", err, attempts)
	}

	attempts = 0
	err = d.InSystemTx(context.Background(), func(context.Context, pgx.Tx) error {
		attempts++
		return &pgconn.PgError{Code: database.SQLStateUniqueViolation}
	})
	if attempts != 1 || !database.IsUniqueViolation(err, "") {
		t.Fatalf("non-retryable error retried: attempts=%d err=%v", attempts, err)
	}
}

func TestCheckMigrations(t *testing.T) {
	t.Parallel()
	d := dbtest.New(t).Open(t, "opsgrid_app", 1)
	if err := d.CheckMigrations(context.Background()); err != nil {
		t.Fatal(err)
	}
}
