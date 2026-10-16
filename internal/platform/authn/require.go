package authn

import (
	"log/slog"
	"net/http"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/httpx"
)

// RequireAuth rejects requests with no Principal (soft middleware left them
// unauthenticated). Place after Middleware on protected route groups.
// GET /v1/meta remains public.
func RequireAuth(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/meta" {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := PrincipalFromContext(r.Context()); !ok {
				httpx.WriteError(w, r, log, apperr.New(
					apperr.CodeAuthenticationRequired,
					"Sign in to continue.",
				))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
