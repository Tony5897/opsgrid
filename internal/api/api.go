// Package api adapts the generated OpenAPI strict-server interface
// (internal/api/gen) to OpsGrid's HTTP platform: request decoding errors and
// handler errors become RFC 9457 problems through httpx.WriteError, and the
// generated routes mount on a standard ServeMux.
//
// Domain modules implement their slice of gen.StrictServerInterface in their
// own http.go adapters; Server composes them.
package api

import (
	"context"
	"log/slog"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/Tony5897/opsgrid/internal/api/gen"
	"github.com/Tony5897/opsgrid/internal/identity"
	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/authn"
	"github.com/Tony5897/opsgrid/internal/platform/buildinfo"
	"github.com/Tony5897/opsgrid/internal/platform/httpx"
	"github.com/Tony5897/opsgrid/internal/tenancy"
)

// Server implements gen.StrictServerInterface.
type Server struct {
	Identity *identity.Service
	Tenancy  *tenancy.Service
}

var _ gen.StrictServerInterface = (*Server)(nil)

// GetMeta returns build information.
func (*Server) GetMeta(context.Context, gen.GetMetaRequestObject) (gen.GetMetaResponseObject, error) {
	return gen.GetMeta200JSONResponse{
		Service: gen.Opsgrid,
		Version: buildinfo.Version,
		Commit:  buildinfo.Short(),
	}, nil
}

// GetMe returns the authenticated caller and active memberships.
func (s *Server) GetMe(ctx context.Context, _ gen.GetMeRequestObject) (gen.GetMeResponseObject, error) {
	p, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return nil, apperr.New(apperr.CodeAuthenticationRequired, "Sign in to continue.")
	}
	me, err := s.Identity.GetMe(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	var email *openapi_types.Email
	if me.User.Email != nil && *me.User.Email != "" {
		e := openapi_types.Email(*me.User.Email)
		email = &e
	}
	out := gen.Me{
		User: gen.MeUser{
			Id:          me.User.ID,
			DisplayName: me.User.DisplayName,
			Email:       email,
		},
		Memberships: make([]gen.MeMembership, 0, len(me.Memberships)),
	}
	for _, m := range me.Memberships {
		out.Memberships = append(out.Memberships, gen.MeMembership{
			OrganizationId: m.OrganizationID,
			Slug:           m.Slug,
			Name:           m.Name,
			Role:           gen.MembershipRole(m.Role),
			Permissions:    m.Permissions,
		})
	}
	return gen.GetMe200JSONResponse(out), nil
}

// GetOrganization returns an org the caller may access.
func (s *Server) GetOrganization(ctx context.Context, req gen.GetOrganizationRequestObject) (gen.GetOrganizationResponseObject, error) {
	p, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return nil, apperr.New(apperr.CodeAuthenticationRequired, "Sign in to continue.")
	}
	org, err := s.Tenancy.GetOrganization(ctx, p.UserID, req.OrgId)
	if err != nil {
		return nil, err
	}
	return gen.GetOrganization200JSONResponse{
		Id:   org.ID,
		Slug: org.Slug,
		Name: org.Name,
	}, nil
}

// Handler mounts the generated routes on mux with OpsGrid error handling.
func Handler(srv gen.StrictServerInterface, mux *http.ServeMux, log *slog.Logger) http.Handler {
	strict := gen.NewStrictHandlerWithOptions(srv, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.WriteError(w, r, log, apperr.Wrap(err, apperr.CodeValidationFailed, "The request is malformed."))
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.WriteError(w, r, log, err)
		},
	})
	return gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			// Parameter binding failures (bad path/query/header values).
			httpx.WriteError(w, r, log, apperr.Wrap(err, apperr.CodeValidationFailed, "A request parameter is invalid."))
		},
	})
}
