//go:build integration

package tenancy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/database"
	"github.com/Tony5897/opsgrid/internal/platform/database/dbtest"
	"github.com/Tony5897/opsgrid/internal/tenancy"
	"github.com/jackc/pgx/v5"
)

func TestGetOrganizationMembershipGate(t *testing.T) {
	t.Parallel()
	tdb := dbtest.New(t)
	db := tdb.Open(t, "opsgrid_app", 4)
	ctx := context.Background()
	svc := tenancy.New(db)

	orgA, orgB := uuid.New(), uuid.New()
	userA, userB := uuid.New(), uuid.New()

	must(t, db.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, u := range []uuid.UUID{userA, userB} {
			if _, err := tx.Exec(ctx, `
				INSERT INTO app.users (id, issuer, subject, email, display_name)
				VALUES ($1, 'https://issuer.example', $2, $3, 'U')`,
				u, "sub-"+u.String(), u.String()+"@example.com"); err != nil {
				return err
			}
		}
		return nil
	}))
	must(t, db.InTenantTx(ctx, database.Scope{OrganizationID: orgA}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO app.organizations (id, name, slug) VALUES ($1, 'A', $2)`,
			orgA, "org-a-"+orgA.String()[:8])
		return err
	}))
	must(t, db.InTenantTx(ctx, database.Scope{OrganizationID: orgB}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO app.organizations (id, name, slug) VALUES ($1, 'B', $2)`,
			orgB, "org-b-"+orgB.String()[:8])
		return err
	}))
	must(t, db.InTenantTx(ctx, database.Scope{OrganizationID: orgA, UserID: userA}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.organization_memberships (organization_id, user_id, role, status)
			VALUES ($1, $2, 'dispatcher', 'active')`, orgA, userA)
		return err
	}))

	// Member of A can read A.
	if _, err := svc.GetOrganization(ctx, userA, orgA); err != nil {
		t.Fatalf("member get own org: %v", err)
	}
	// Cascade user cannot read Northstar org → TENANT_NOT_FOUND (404 semantics).
	_, err := svc.GetOrganization(ctx, userA, orgB)
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.CodeTenantNotFound {
		t.Fatalf("cross-tenant get: %v, want TENANT_NOT_FOUND", err)
	}

	// Removed member denied on next request.
	must(t, db.InTenantTx(ctx, database.Scope{OrganizationID: orgA, UserID: userA}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE app.organization_memberships SET status = 'removed', updated_at = now()
			WHERE organization_id = $1 AND user_id = $2`, orgA, userA)
		return err
	}))
	_, err = svc.GetOrganization(ctx, userA, orgA)
	if !errors.As(err, &ae) || ae.Code != apperr.CodeTenantNotFound {
		t.Fatalf("removed member: %v, want TENANT_NOT_FOUND", err)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
