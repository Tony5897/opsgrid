//go:build integration

package identity_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Tony5897/opsgrid/internal/identity"
	"github.com/Tony5897/opsgrid/internal/platform/database"
	"github.com/Tony5897/opsgrid/internal/platform/database/dbtest"
)

func TestGetMeReturnsOnlyOwnMemberships(t *testing.T) {
	t.Parallel()
	tdb := dbtest.New(t)
	db := tdb.Open(t, "opsgrid_app", 4)
	ctx := context.Background()
	svc := identity.New(db)

	orgA, orgB := uuid.New(), uuid.New()
	userA, userB := uuid.New(), uuid.New()

	must(t, db.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, u := range []struct {
			id   uuid.UUID
			name string
		}{
			{userA, "Alex"},
			{userB, "Dana"},
		} {
			if _, err := tx.Exec(ctx, `
				INSERT INTO app.users (id, issuer, subject, email, display_name)
				VALUES ($1, 'https://issuer.example', $2, $3, $4)`,
				u.id, "sub-"+u.id.String(), u.id.String()+"@example.com", u.name); err != nil {
				return err
			}
		}
		return nil
	}))
	for _, org := range []struct {
		id   uuid.UUID
		slug string
		name string
	}{
		{orgA, "cascade-" + orgA.String()[:8], "Cascade"},
		{orgB, "north-" + orgB.String()[:8], "Northstar"},
	} {
		must(t, db.InTenantTx(ctx, database.Scope{OrganizationID: org.id}, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO app.organizations (id, name, slug) VALUES ($1, $2, $3)`,
				org.id, org.name, org.slug)
			return err
		}))
	}
	// userA in both orgs (alex dual-membership pattern).
	must(t, db.InTenantTx(ctx, database.Scope{OrganizationID: orgA, UserID: userA}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.organization_memberships (organization_id, user_id, role)
			VALUES ($1, $2, 'dispatcher')`, orgA, userA)
		return err
	}))
	must(t, db.InTenantTx(ctx, database.Scope{OrganizationID: orgB, UserID: userA}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.organization_memberships (organization_id, user_id, role)
			VALUES ($1, $2, 'viewer')`, orgB, userA)
		return err
	}))
	must(t, db.InTenantTx(ctx, database.Scope{OrganizationID: orgA, UserID: userB}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.organization_memberships (organization_id, user_id, role)
			VALUES ($1, $2, 'owner')`, orgA, userB)
		return err
	}))

	me, err := svc.GetMe(ctx, userA)
	if err != nil {
		t.Fatal(err)
	}
	if me.User.DisplayName != "Alex" {
		t.Fatalf("display = %q", me.User.DisplayName)
	}
	if len(me.Memberships) != 2 {
		t.Fatalf("memberships = %d, want 2", len(me.Memberships))
	}
	roles := map[string]string{}
	perms := map[string][]string{}
	for _, m := range me.Memberships {
		roles[m.Slug] = string(m.Role)
		perms[m.Slug] = m.Permissions
		if len(m.Permissions) == 0 {
			t.Fatalf("empty permissions for %s", m.Slug)
		}
	}
	cascadeSlug := "cascade-" + orgA.String()[:8]
	northSlug := "north-" + orgB.String()[:8]
	if roles[cascadeSlug] != "dispatcher" {
		t.Fatalf("cascade role = %v", roles)
	}
	if roles[northSlug] != "viewer" {
		t.Fatalf("north role = %v", roles)
	}
	if contains(perms[cascadeSlug], "platform.operate") || contains(perms[northSlug], "platform.operate") {
		t.Fatal("non-owner must not have platform.operate")
	}
	if len(perms[cascadeSlug]) <= len(perms[northSlug]) {
		t.Fatalf("dispatcher should have more permissions than viewer: %d vs %d",
			len(perms[cascadeSlug]), len(perms[northSlug]))
	}

	meB, err := svc.GetMe(ctx, userB)
	if err != nil {
		t.Fatal(err)
	}
	if len(meB.Memberships) != 1 {
		t.Fatalf("userB memberships = %d, want 1", len(meB.Memberships))
	}
	if !contains(meB.Memberships[0].Permissions, "platform.operate") {
		t.Fatal("owner must include platform.operate")
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
