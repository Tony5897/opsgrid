// Package seed loads deterministic demo tenants for local development (G1).
// It runs as opsgrid_app and uses InTenantTx — never BYPASSRLS.
package seed

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Tony5897/opsgrid/internal/platform/database"
)

// DemoIssuer matches OIDC_ISSUER_URL / Keycloak frontend hostname.
const DemoIssuer = "http://localhost:8180/realms/opsgrid"

// Fixed demo identities (pinned Keycloak user ids = OIDC sub).
var (
	OrgCascade   = uuid.MustParse("01939a00-0001-7000-8000-000000000001")
	OrgNorthstar = uuid.MustParse("01939a00-0002-7000-8000-000000000002")

	UserOwen  = uuid.MustParse("01939a10-0001-7000-8000-000000000001")
	UserDana  = uuid.MustParse("01939a10-0002-7000-8000-000000000002")
	UserMaria = uuid.MustParse("01939a10-0003-7000-8000-000000000003")
	UserJames = uuid.MustParse("01939a10-0004-7000-8000-000000000004")
	UserNina  = uuid.MustParse("01939a10-0005-7000-8000-000000000005")
	UserAlex  = uuid.MustParse("01939a10-0006-7000-8000-000000000006")

	// Keycloak subjects (= realm user "id" fields).
	SubOwen  = "01939b00-0001-7000-8000-000000000001"
	SubDana  = "01939b00-0002-7000-8000-000000000002"
	SubMaria = "01939b00-0003-7000-8000-000000000003"
	SubJames = "01939b00-0004-7000-8000-000000000004"
	SubNina  = "01939b00-0005-7000-8000-000000000005"
	SubAlex  = "01939b00-0006-7000-8000-000000000006"
)

type userSpec struct {
	id          uuid.UUID
	subject     string
	email       string
	displayName string
}

type locSpec struct {
	id       uuid.UUID
	name, tz string
}

type membershipSpec struct {
	org, user uuid.UUID
	role      string
}

// Run upserts Cascade / Northstar orgs, locations, demo users and memberships.
func Run(ctx context.Context, db *database.DB, log *slog.Logger) error {
	users := []userSpec{
		{UserOwen, SubOwen, "owen@cascade.example", "Owen Park"},
		{UserDana, SubDana, "dana@cascade.example", "Dana Reyes"},
		{UserMaria, SubMaria, "maria@cascade.example", "Maria Chen"},
		{UserJames, SubJames, "james@cascade.example", "James Okafor"},
		{UserNina, SubNina, "nina@northstar.example", "Nina Larsen"},
		{UserAlex, SubAlex, "alex@opsgrid.example", "Alex Morgan"},
	}
	for _, u := range users {
		if err := upsertUser(ctx, db, u); err != nil {
			return fmt.Errorf("user %s: %w", u.email, err)
		}
	}

	if err := upsertOrg(ctx, db, OrgCascade, "Cascade Facilities", "cascade-facilities", []locSpec{
		{uuid.MustParse("01939a20-0001-7000-8000-000000000001"), "Portland", "America/Los_Angeles"},
		{uuid.MustParse("01939a20-0002-7000-8000-000000000002"), "Salem", "America/Los_Angeles"},
		{uuid.MustParse("01939a20-0003-7000-8000-000000000003"), "Eugene", "America/Los_Angeles"},
	}); err != nil {
		return err
	}
	if err := upsertOrg(ctx, db, OrgNorthstar, "Northstar Mechanical", "northstar-mechanical", []locSpec{
		{uuid.MustParse("01939a20-0011-7000-8000-000000000011"), "Seattle", "America/Los_Angeles"},
		{uuid.MustParse("01939a20-0012-7000-8000-000000000012"), "Tacoma", "America/Los_Angeles"},
	}); err != nil {
		return err
	}

	memberships := []membershipSpec{
		{OrgCascade, UserOwen, "owner"},
		{OrgCascade, UserDana, "dispatcher"},
		{OrgCascade, UserMaria, "technician"},
		{OrgCascade, UserJames, "technician"},
		{OrgCascade, UserAlex, "dispatcher"},
		{OrgNorthstar, UserNina, "dispatcher"},
		{OrgNorthstar, UserAlex, "viewer"},
	}
	for _, m := range memberships {
		if err := upsertMembership(ctx, db, m); err != nil {
			return fmt.Errorf("membership %s/%s: %w", m.org, m.user, err)
		}
	}

	log.Info("demo seed complete",
		slog.String("cascade", OrgCascade.String()),
		slog.String("northstar", OrgNorthstar.String()),
		slog.Int("users", len(users)),
		slog.Int("memberships", len(memberships)))
	return nil
}

func upsertUser(ctx context.Context, db *database.DB, u userSpec) error {
	return db.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.users (id, issuer, subject, email, display_name)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (issuer, subject) DO UPDATE
			SET email = EXCLUDED.email,
			    display_name = EXCLUDED.display_name,
			    updated_at = now()`,
			u.id, DemoIssuer, u.subject, u.email, u.displayName)
		return err
	})
}

func upsertOrg(ctx context.Context, db *database.DB, orgID uuid.UUID, name, slug string, locs []locSpec) error {
	scope := database.Scope{OrganizationID: orgID}
	if err := db.InTenantTx(ctx, scope, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.organizations (id, name, slug)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE
			SET name = EXCLUDED.name, slug = EXCLUDED.slug, updated_at = now()`,
			orgID, name, slug)
		return err
	}); err != nil {
		return fmt.Errorf("org %s: %w", slug, err)
	}
	for _, loc := range locs {
		if err := db.InTenantTx(ctx, scope, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO app.locations (id, organization_id, name, timezone)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (id) DO UPDATE
				SET name = EXCLUDED.name, timezone = EXCLUDED.timezone, updated_at = now()`,
				loc.id, orgID, loc.name, loc.tz)
			return err
		}); err != nil {
			return fmt.Errorf("location %s: %w", loc.name, err)
		}
	}
	return nil
}

func upsertMembership(ctx context.Context, db *database.DB, m membershipSpec) error {
	return db.InTenantTx(ctx, database.Scope{OrganizationID: m.org, UserID: m.user}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO app.organization_memberships (organization_id, user_id, role, status)
			VALUES ($1, $2, $3, 'active')
			ON CONFLICT (organization_id, user_id) DO UPDATE
			SET role = EXCLUDED.role, status = 'active', updated_at = now()`,
			m.org, m.user, m.role)
		return err
	})
}
