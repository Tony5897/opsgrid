package tenancy

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/authz"
	"github.com/Tony5897/opsgrid/internal/platform/database"
)

// Organization is a tenant root visible to an active member.
type Organization struct {
	ID   uuid.UUID
	Slug string
	Name string
}

// Membership is an active org membership for authorization context.
type Membership struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           authz.Role
}

// Service resolves memberships and org records under RLS.
type Service struct {
	db *database.DB
}

// New returns a tenancy Service.
func New(db *database.DB) *Service { return &Service{db: db} }

// ActiveMembership returns the caller's active membership in orgID, or
// TENANT_NOT_FOUND when missing/removed (404, not 403).
func (s *Service) ActiveMembership(ctx context.Context, userID, orgID uuid.UUID) (Membership, error) {
	var m Membership
	err := s.db.InUserTx(ctx, userID, func(ctx context.Context, tx pgx.Tx) error {
		var role string
		err := tx.QueryRow(ctx, `
			SELECT role FROM app.organization_memberships
			WHERE organization_id = $1 AND user_id = $2 AND status = 'active'`,
			orgID, userID).Scan(&role)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.New(apperr.CodeTenantNotFound, "Organization not found.")
		}
		if err != nil {
			return err
		}
		parsed, ok := authz.ParseRole(role)
		if !ok {
			return fmt.Errorf("unknown role %q", role)
		}
		m = Membership{OrganizationID: orgID, UserID: userID, Role: parsed}
		return nil
	})
	return m, err
}

// GetOrganization returns the org when the caller is an active member.
func (s *Service) GetOrganization(ctx context.Context, userID, orgID uuid.UUID) (Organization, error) {
	if _, err := s.ActiveMembership(ctx, userID, orgID); err != nil {
		return Organization{}, err
	}
	var o Organization
	err := s.db.InTenantTx(ctx, database.Scope{OrganizationID: orgID, UserID: userID}, func(ctx context.Context, tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT id, slug::text, name FROM app.organizations WHERE id = $1`, orgID).
			Scan(&o.ID, &o.Slug, &o.Name)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.New(apperr.CodeTenantNotFound, "Organization not found.")
		}
		return err
	})
	return o, err
}
