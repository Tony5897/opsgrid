package authn

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Middleware loads a live session from the __Host-opsgrid_session cookie and
// attaches a Principal to the request context. Missing or expired cookies are
// not errors; protected routes enforce authentication separately.
func Middleware(store *Store, log *slog.Logger, idleTTL time.Duration, now func() time.Time) func(http.Handler) http.Handler {
	if now == nil {
		now = time.Now
	}
	if idleTTL == 0 {
		idleTTL = 2 * time.Hour
	}
	const touchMin = time.Minute
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(SessionCookieName)
			if err != nil || c.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			hash := HashSessionID(c.Value)
			t := now()
			sess, err := store.GetLiveSession(r.Context(), hash, t)
			if err != nil {
				if errors.Is(err, ErrSessionNotFound) {
					ClearSessionCookie(w)
				} else {
					// Do not clear the cookie on transient DB errors — that would
					// force a re-login for an infrastructure blip.
					log.WarnContext(r.Context(), "session lookup failed", slog.Any("error", err))
				}
				next.ServeHTTP(w, r)
				return
			}
			// Best-effort sliding idle window; failures must not block the request.
			if err := store.TouchSession(r.Context(), hash, t, idleTTL, touchMin); err != nil {
				log.WarnContext(r.Context(), "session touch failed", slog.Any("error", err))
			}
			ctx := ContextWithPrincipal(r.Context(), Principal{UserID: sess.UserID, SessionHash: hash})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
