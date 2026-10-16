package tenancy

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/authn"
	"github.com/Tony5897/opsgrid/internal/platform/authz"
	"github.com/Tony5897/opsgrid/internal/platform/httpx"
)

type ctxKey int

const membershipKey ctxKey = 1

// MembershipFromContext returns the org membership attached by ResolveOrg.
func MembershipFromContext(ctx context.Context) (Membership, bool) {
	m, ok := ctx.Value(membershipKey).(Membership)
	return m, ok
}

// ResolveOrg re-reads active membership for path {orgId} on every request.
// Missing/removed membership → 404 TENANT_NOT_FOUND.
func ResolveOrg(svc *Service, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.PathValue("orgId")
			if raw == "" {
				next.ServeHTTP(w, r)
				return
			}
			orgID, err := uuid.Parse(raw)
			if err != nil {
				httpx.WriteError(w, r, log, apperr.New(apperr.CodeValidationFailed, "Invalid organization id."))
				return
			}
			p, ok := authn.PrincipalFromContext(r.Context())
			if !ok {
				httpx.WriteError(w, r, log, apperr.New(apperr.CodeAuthenticationRequired, "Sign in to continue."))
				return
			}
			m, err := svc.ActiveMembership(r.Context(), p.UserID, orgID)
			if err != nil {
				httpx.WriteError(w, r, log, err)
				return
			}
			ctx := context.WithValue(r.Context(), membershipKey, m)
			ctx = context.WithValue(ctx, authzPrincipalKey{}, authz.Principal{
				UserID:         m.UserID,
				OrganizationID: m.OrganizationID,
				Role:           m.Role,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type authzPrincipalKey struct{}

// AuthzPrincipalFromContext returns the authz principal set by ResolveOrg.
func AuthzPrincipalFromContext(ctx context.Context) (authz.Principal, bool) {
	p, ok := ctx.Value(authzPrincipalKey{}).(authz.Principal)
	return p, ok
}
