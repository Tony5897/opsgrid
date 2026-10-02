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

	"github.com/Tony5897/opsgrid/internal/api/gen"
	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/buildinfo"
	"github.com/Tony5897/opsgrid/internal/platform/httpx"
)

// Server implements gen.StrictServerInterface.
type Server struct{}

var _ gen.StrictServerInterface = (*Server)(nil)

// GetMeta returns build information.
func (*Server) GetMeta(context.Context, gen.GetMetaRequestObject) (gen.GetMetaResponseObject, error) {
	return gen.GetMeta200JSONResponse{
		Service: gen.Opsgrid,
		Version: buildinfo.Version,
		Commit:  buildinfo.Short(),
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
