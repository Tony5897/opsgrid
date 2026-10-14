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

// Real tenant tables (migrations 00002–00003): same-tenant OK, cross-tenant
// denied, missing context fails closed.
func TestOrganizationsRLSFailsClosed(t *testing.T) {
	t.Parallel()
	tdb := dbtest.New(t)
	app := tdb.Open(t, "opsgrid_app", 2)
	ctx := context.Background()
	orgA, orgB := uuid.New(), uuid.New()

	insertOrg := func(scope, id uuid.UUID, slug string) error {
		return app.InTenantTx(ctx, database.Scope{OrganizationID: scope}, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx,
				`INSERT INTO app.organizations (id, name, slug) VALUES ($1, $2, $3)`,
				id, "Org "+slug, slug)
			return err
		})
	}
	insertLoc := func(scope, org, loc uuid.UUID) error {
		return app.InTenantTx(ctx, database.Scope{OrganizationID: scope}, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx,
				`INSERT INTO app.locations (id, organization_id, name, timezone) VALUES ($1, $2, 'HQ', 'America/Los_Angeles')`,
				loc, org)
			return err
		})
	}
	countOrgs := func(scope *uuid.UUID) int {
		var n int
		run := func(ctx context.Context, tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM app.organizations`).Scan(&n)
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
	countLocs := func(scope uuid.UUID) int {
		var n int
		err := app.InTenantTx(ctx, database.Scope{OrganizationID: scope}, func(ctx context.Context, tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM app.locations`).Scan(&n)
		})
		if err != nil {
			t.Fatal(err)
		}
		return n
	}

	if err := insertOrg(orgA, orgA, "cascade-"+orgA.String()[:8]); err != nil {
		t.Fatalf("insert org A: %v", err)
	}
	if err := insertOrg(orgB, orgB, "northstar-"+orgB.String()[:8]); err != nil {
		t.Fatalf("insert org B: %v", err)
	}

	// WITH CHECK: cannot insert an organization row under another tenant's scope.
	err := insertOrg(orgA, orgB, "spoof-"+uuid.NewString()[:8])
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != database.SQLStateInsufficientPriv {
		t.Fatalf("cross-tenant org insert: err = %v, want SQLSTATE 42501", err)
	}

	locA, locB := uuid.New(), uuid.New()
	if err := insertLoc(orgA, orgA, locA); err != nil {
		t.Fatalf("insert location A: %v", err)
	}
	if err := insertLoc(orgB, orgB, locB); err != nil {
		t.Fatalf("insert location B: %v", err)
	}
	err = insertLoc(orgA, orgB, uuid.New())
	if !errors.As(err, &pgErr) || pgErr.Code != database.SQLStateInsufficientPriv {
		t.Fatalf("cross-tenant location insert: err = %v, want SQLSTATE 42501", err)
	}

	if n := countOrgs(&orgA); n != 1 {
		t.Errorf("tenant A sees %d orgs, want 1", n)
	}
	if n := countOrgs(&orgB); n != 1 {
		t.Errorf("tenant B sees %d orgs, want 1", n)
	}
	if n := countOrgs(nil); n != 0 {
		t.Errorf("no tenant context sees %d orgs, want 0 (fail closed)", n)
	}
	if n := countLocs(orgA); n != 1 {
		t.Errorf("tenant A sees %d locations, want 1", n)
	}
	if n := countLocs(orgB); n != 1 {
		t.Errorf("tenant B sees %d locations, want 1", n)
	}

	// Counter under tenant context.
	err = app.InTenantTx(ctx, database.Scope{OrganizationID: orgA}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO app.organization_counters (organization_id, name, value) VALUES ($1, 'work_order', 0)`,
			orgA)
		return err
	})
	if err != nil {
		t.Fatalf("insert counter: %v", err)
	}
	err = app.InTenantTx(ctx, database.Scope{OrganizationID: orgA}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO app.organization_counters (organization_id, name, value) VALUES ($1, 'work_order', 0)`,
			orgB)
		return err
	})
	if !errors.As(err, &pgErr) || pgErr.Code != database.SQLStateInsufficientPriv {
		t.Fatalf("cross-tenant counter insert: err = %v, want SQLSTATE 42501", err)
	}
}

func TestMembershipsVisibleInUserTx(t *testing.T) {
	t.Parallel()
	tdb := dbtest.New(t)
	app := tdb.Open(t, "opsgrid_app", 2)
	ctx := context.Background()
	orgID, userA, userB := uuid.New(), uuid.New(), uuid.New()

	err := app.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, u := range []struct {
			id, sub, email, name string
		}{
			{userA.String(), "sub-a-" + userA.String()[:8], "a@example.com", "User A"},
			{userB.String(), "sub-b-" + userB.String()[:8], "b@example.com", "User B"},
		} {
			if _, err := tx.Exec(ctx, `
				INSERT INTO app.users (id, issuer, subject, email, display_name)
				VALUES ($1::uuid, 'https://issuer.example', $2, $3, $4)`,
				u.id, u.sub, u.email, u.name); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}

	err = app.InTenantTx(ctx, database.Scope{OrganizationID: orgID, UserID: userA}, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`INSERT INTO app.organizations (id, name, slug) VALUES ($1, 'Cascade', $2)`,
			orgID, "cascade-"+orgID.String()[:8]); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.organization_memberships (organization_id, user_id, role)
			VALUES ($1, $2, 'dispatcher'), ($1, $3, 'viewer')`,
			orgID, userA, userB); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed org + memberships: %v", err)
	}

	// InUserTx: each user sees only their own membership rows.
	countMine := func(user uuid.UUID) int {
		var n int
		err := app.InUserTx(ctx, user, func(ctx context.Context, tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM app.organization_memberships`).Scan(&n)
		})
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := countMine(userA); n != 1 {
		t.Errorf("user A sees %d memberships, want 1", n)
	}
	if n := countMine(userB); n != 1 {
		t.Errorf("user B sees %d memberships, want 1", n)
	}

	// InTenantTx: org scope sees both members.
	var n int
	err = app.InTenantTx(ctx, database.Scope{OrganizationID: orgID, UserID: userA}, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM app.organization_memberships`).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("org scope sees %d memberships, want 2", n)
	}

	// No context: fail closed.
	err = app.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM app.organization_memberships`).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("no context sees %d memberships, want 0", n)
	}
}

// Schema invariants for tenant tables introduced in Gate 1 (ledger / ADR-003/004).
func TestSchemaTenantTablesHaveRLS(t *testing.T) {
	t.Parallel()
	tdb := dbtest.New(t)
	ctx := context.Background()
	su, err := pgx.Connect(ctx, tdb.SuperURL())
	if err != nil {
		t.Fatal(err)
	}
	defer su.Close(ctx)

	// organizations: FORCE RLS + a policy (tenant root uses id, not organization_id).
	var forced, hasPolicy bool
	err = su.QueryRow(ctx, `
		SELECT c.relrowsecurity AND c.relforcerowsecurity,
		       EXISTS (
		         SELECT 1 FROM pg_policy p
		         WHERE p.polrelid = c.oid
		       )
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'app' AND c.relname = 'organizations'`).Scan(&forced, &hasPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if !forced || !hasPolicy {
		t.Errorf("organizations: forced_rls=%v has_policy=%v, want true/true", forced, hasPolicy)
	}

	// Every app table with an organization_id column: FORCE RLS, ≥1 policy,
	// and a unique/primary constraint that leads with organization_id.
	rows, err := su.Query(ctx, `
		SELECT c.relname,
		       c.relrowsecurity AND c.relforcerowsecurity AS forced,
		       EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid = c.oid) AS has_policy,
		       EXISTS (
		         SELECT 1
		         FROM pg_constraint con
		         WHERE con.conrelid = c.oid
		           AND con.contype IN ('p', 'u')
		           AND (
		             SELECT a.attname
		             FROM unnest(con.conkey) WITH ORDINALITY AS k(attnum, ord)
		             JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = k.attnum
		             ORDER BY k.ord
		             LIMIT 1
		           ) = 'organization_id'
		       ) AS org_leading_unique
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attname = 'organization_id' AND NOT a.attisdropped
		WHERE n.nspname = 'app' AND c.relkind = 'r'
		ORDER BY c.relname`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	seen := 0
	for rows.Next() {
		var name string
		var forced, hasPolicy, orgUnique bool
		if err := rows.Scan(&name, &forced, &hasPolicy, &orgUnique); err != nil {
			t.Fatal(err)
		}
		seen++
		if !forced {
			t.Errorf("%s: RLS not enabled+forced", name)
		}
		if !hasPolicy {
			t.Errorf("%s: missing RLS policy", name)
		}
		if !orgUnique {
			t.Errorf("%s: missing unique/primary constraint leading with organization_id", name)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	// locations, organization_memberships, organization_counters (not organizations).
	if seen < 3 {
		t.Fatalf("found %d tables with organization_id, want at least 3", seen)
	}

	// users and sessions must remain without RLS (identity bootstrap).
	for _, table := range []string{"users", "sessions"} {
		var rls bool
		err := su.QueryRow(ctx, `
			SELECT c.relrowsecurity
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'app' AND c.relname = $1`, table).Scan(&rls)
		if err != nil {
			t.Fatal(err)
		}
		if rls {
			t.Errorf("%s: unexpected RLS enabled (identity tables must stay global)", table)
		}
	}
}
