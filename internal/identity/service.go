package identity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Tony5897/opsgrid/internal/platform/authz"
	"github.com/Tony5897/opsgrid/internal/platform/database"
)

// Me is the /v1/me response payload (before OpenAPI DTO mapping).
type Me struct {
	User        User
	Memberships []Membership
}

// User is the authenticated account profile.
type User struct {
	ID          uuid.UUID
	Email       *string
	DisplayName string
}

// Membership is an active org membership with expanded permissions.
type Membership struct {
	OrganizationID uuid.UUID
	Slug           string
	Name           string
	Role           authz.Role
	Permissions    []string
}

// Service loads caller identity under RLS-safe transactions.
type Service struct {
	db *database.DB
}

// New returns an identity Service.
func New(db *database.DB) *Service { return &Service{db: db} }

type membershipRow struct {
	org  uuid.UUID
	role authz.Role
}

// GetMe loads the user and active memberships for userID.
// Memberships are loaded in InUserTx; org name/slug hydrate uses separate
// InTenantTx calls so transactions are never nested (ADR-004).
func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (Me, error) {
	var me Me
	var pending []membershipRow
	err := s.db.InUserTx(ctx, userID, func(ctx context.Context, tx pgx.Tx) error {
		var email *string
		var display string
		if err := tx.QueryRow(ctx, `
			SELECT email::text, COALESCE(display_name, '')
			FROM app.users WHERE id = $1`, userID).Scan(&email, &display); err != nil {
			return fmt.Errorf("load user: %w", err)
		}
		me.User = User{ID: userID, Email: email, DisplayName: display}

		rows, err := tx.Query(ctx, `
			SELECT organization_id, role
			FROM app.organization_memberships
			WHERE user_id = $1 AND status = 'active'
			ORDER BY organization_id`, userID)
		if err != nil {
			return fmt.Errorf("list memberships: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var org uuid.UUID
			var roleStr string
			if err := rows.Scan(&org, &roleStr); err != nil {
				return err
			}
			role, ok := authz.ParseRole(roleStr)
			if !ok {
				return fmt.Errorf("unknown role %q", roleStr)
			}
			pending = append(pending, membershipRow{org: org, role: role})
		}
		return rows.Err()
	})
	if err != nil {
		return Me{}, err
	}

	me.Memberships = make([]Membership, 0, len(pending))
	for _, r := range pending {
		org, err := s.loadOrg(ctx, r.org)
		if err != nil {
			return Me{}, err
		}
		me.Memberships = append(me.Memberships, Membership{
			OrganizationID: r.org,
			Slug:           org.Slug,
			Name:           org.Name,
			Role:           r.role,
			Permissions:    authz.PermissionStrings(r.role),
		})
	}
	return me, nil
}

type orgRow struct {
	Name, Slug string
}

func (s *Service) loadOrg(ctx context.Context, orgID uuid.UUID) (orgRow, error) {
	var o orgRow
	err := s.db.InTenantTx(ctx, database.Scope{OrganizationID: orgID}, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT name, slug::text FROM app.organizations WHERE id = $1`, orgID).Scan(&o.Name, &o.Slug)
	})
	if err != nil {
		return orgRow{}, fmt.Errorf("hydrate org %s: %w", orgID, err)
	}
	return o, nil
}
