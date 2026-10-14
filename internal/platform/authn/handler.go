package authn

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/httpx"
)

// OIDCClient is the IdP surface used by auth handlers (real: *Provider).
type OIDCClient interface {
	Warm(ctx context.Context) error
	AuthCodeURL(state, nonce, challenge string) string
	Exchange(ctx context.Context, code, verifier, nonce string) (TokenBundle, error)
	EndSessionURL(ctx context.Context, postLogoutRedirect string) (string, error)
	VerifyLogoutToken(ctx context.Context, raw string) (LogoutClaims, error)
}

// Options configure the auth HTTP handlers.
type Options struct {
	Provider     OIDCClient
	Store        *Store
	Sealer       *Sealer
	Log          *slog.Logger
	PublicOrigin string // SPA origin for returnTo allowlist and post-logout redirect
	AbsoluteTTL  time.Duration
	IdleTTL      time.Duration
	Now          func() time.Time
}

// Handler serves /auth/login, /auth/callback, /auth/logout, /auth/backchannel-logout.
func Handler(opts Options) http.Handler {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.AbsoluteTTL == 0 {
		opts.AbsoluteTTL = 12 * time.Hour
	}
	if opts.IdleTTL == 0 {
		opts.IdleTTL = 2 * time.Hour
	}
	mux := http.NewServeMux()
	h := &handlers{opts: opts}
	mux.HandleFunc("GET /auth/login", h.login)
	mux.HandleFunc("GET /auth/callback", h.callback)
	// POST only: GET would bypass CSRF (safe-method exemption) and enable
	// cross-site logout.
	mux.HandleFunc("POST /auth/logout", h.logout)
	mux.HandleFunc("POST /auth/backchannel-logout", h.backchannel)
	return mux
}

type handlers struct {
	opts Options
}

func (h *handlers) login(w http.ResponseWriter, r *http.Request) {
	if err := h.opts.Provider.Warm(r.Context()); err != nil {
		h.opts.Log.WarnContext(r.Context(), "oidc unavailable", slog.Any("error", err))
		httpx.WriteError(w, r, h.opts.Log, apperr.New(apperr.CodeDependencyUnavailable, "Identity provider is unavailable."))
		return
	}
	returnTo := sanitizeReturnTo(r.URL.Query().Get("returnTo"), h.opts.PublicOrigin)
	hs, challenge, err := newHandshake(returnTo, h.opts.Now())
	if err != nil {
		httpx.WriteError(w, r, h.opts.Log, apperr.Wrap(err, apperr.CodeInternal, "Could not start login."))
		return
	}
	sealed, err := h.opts.Sealer.sealHandshake(hs)
	if err != nil {
		httpx.WriteError(w, r, h.opts.Log, apperr.Wrap(err, apperr.CodeInternal, "Could not start login."))
		return
	}
	SetOIDCCookie(w, sealed, oidcHandshakeTTL)
	// AuthCodeURL is built from configured IdP discovery endpoints, not user input.
	http.Redirect(w, r, h.opts.Provider.AuthCodeURL(hs.State, hs.Nonce, challenge), http.StatusFound) //nolint:gosec // G710: IdP URL from config
}

func (h *handlers) callback(w http.ResponseWriter, r *http.Request) {
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		ClearOIDCCookie(w)
		httpx.WriteError(w, r, h.opts.Log, apperr.New(apperr.CodeAuthenticationRequired,
			"Sign-in was cancelled or rejected by the identity provider."))
		return
	}
	c, err := r.Cookie(OIDCCookieName)
	if err != nil || c.Value == "" {
		ClearOIDCCookie(w)
		httpx.WriteError(w, r, h.opts.Log, apperr.New(apperr.CodeAuthenticationRequired, "Login session expired. Try again."))
		return
	}
	hs, err := h.opts.Sealer.openHandshake(c.Value, h.opts.Now())
	ClearOIDCCookie(w)
	if err != nil {
		httpx.WriteError(w, r, h.opts.Log, apperr.New(apperr.CodeAuthenticationRequired, "Login session expired. Try again."))
		return
	}
	gotState := r.URL.Query().Get("state")
	if subtle.ConstantTimeCompare([]byte(gotState), []byte(hs.State)) != 1 {
		httpx.WriteError(w, r, h.opts.Log, apperr.New(apperr.CodeAuthenticationRequired, "Invalid login state."))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		httpx.WriteError(w, r, h.opts.Log, apperr.New(apperr.CodeAuthenticationRequired, "Missing authorization code."))
		return
	}
	bundle, err := h.opts.Provider.Exchange(r.Context(), code, hs.CodeVerifier, hs.Nonce)
	if err != nil {
		h.opts.Log.WarnContext(r.Context(), "oidc exchange failed", slog.Any("error", err))
		httpx.WriteError(w, r, h.opts.Log, apperr.New(apperr.CodeAuthenticationRequired, "Sign-in failed."))
		return
	}
	if bundle.RefreshToken == "" {
		httpx.WriteError(w, r, h.opts.Log, apperr.New(apperr.CodeDependencyUnavailable, "Identity provider did not return a refresh token."))
		return
	}
	userID, err := h.opts.Store.UpsertUser(r.Context(), UserUpsert{
		Issuer:      bundle.Issuer,
		Subject:     bundle.Subject,
		Email:       bundle.Email,
		DisplayName: bundle.DisplayName,
	})
	if err != nil {
		httpx.WriteError(w, r, h.opts.Log, apperr.Wrap(err, apperr.CodeInternal, "Could not create user."))
		return
	}
	// Rotate: delete any prior session bound to this browser cookie.
	if prev, err := r.Cookie(SessionCookieName); err == nil && prev.Value != "" {
		_ = h.opts.Store.DeleteSession(r.Context(), HashSessionID(prev.Value))
	}
	rawID, idHash, err := NewSessionID()
	if err != nil {
		httpx.WriteError(w, r, h.opts.Log, apperr.Wrap(err, apperr.CodeInternal, "Could not create session."))
		return
	}
	enc, err := h.opts.Sealer.SealBinary([]byte(bundle.RefreshToken))
	if err != nil {
		httpx.WriteError(w, r, h.opts.Log, apperr.Wrap(err, apperr.CodeInternal, "Could not create session."))
		return
	}
	now := h.opts.Now()
	if err := h.opts.Store.CreateSession(r.Context(), CreateSessionParams{
		UserID:            userID,
		IDHash:            idHash,
		AbsoluteExpiresAt: now.Add(h.opts.AbsoluteTTL),
		IdleExpiresAt:     now.Add(h.opts.IdleTTL),
		RefreshTokenEnc:   enc,
		RefreshTokenKeyID: h.opts.Sealer.KeyID(),
		IP:                clientIP(r),
		UserAgent:         truncate(r.UserAgent(), 512),
	}); err != nil {
		httpx.WriteError(w, r, h.opts.Log, apperr.Wrap(err, apperr.CodeInternal, "Could not create session."))
		return
	}
	ClearSessionCookie(w)
	SetSessionCookie(w, rawID, h.opts.AbsoluteTTL)
	http.Redirect(w, r, sanitizeReturnTo(hs.ReturnTo, h.opts.PublicOrigin), http.StatusFound)
}

func (h *handlers) logout(w http.ResponseWriter, r *http.Request) {
	if h.opts.Store != nil {
		if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" {
			_ = h.opts.Store.DeleteSession(r.Context(), HashSessionID(c.Value))
		}
	}
	ClearSessionCookie(w)
	ClearOIDCCookie(w)
	postLogout := strings.TrimRight(h.opts.PublicOrigin, "/") + "/"
	endURL, err := h.opts.Provider.EndSessionURL(r.Context(), postLogout)
	if err != nil {
		// Local session already cleared; IdP outage must not block logout.
		http.Redirect(w, r, postLogout, http.StatusFound)
		return
	}
	http.Redirect(w, r, endURL, http.StatusFound)
}

func (h *handlers) backchannel(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	raw := r.FormValue("logout_token")
	if raw == "" {
		http.Error(w, "missing logout_token", http.StatusBadRequest)
		return
	}
	claims, err := h.opts.Provider.VerifyLogoutToken(r.Context(), raw)
	if err != nil {
		h.opts.Log.WarnContext(r.Context(), "backchannel logout rejected", slog.Any("error", err))
		http.Error(w, "invalid logout_token", http.StatusBadRequest)
		return
	}
	n, err := h.opts.Store.DeleteSessionsForSubject(r.Context(), claims.Issuer, claims.Subject)
	if err != nil {
		h.opts.Log.ErrorContext(r.Context(), "backchannel logout delete failed", slog.Any("error", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.opts.Log.InfoContext(r.Context(), "backchannel logout",
		slog.String("subject", claims.Subject), slog.Int64("sessions_deleted", n))
	w.WriteHeader(http.StatusOK)
}

// sanitizeReturnTo allows only same-app relative paths (open-redirect safe).
func sanitizeReturnTo(raw, publicOrigin string) string {
	const fallback = "/"
	if raw == "" {
		return fallback
	}
	if u, err := url.Parse(raw); err == nil && u.IsAbs() {
		origin, err := url.Parse(publicOrigin)
		if err != nil {
			return fallback
		}
		if !strings.EqualFold(u.Scheme, origin.Scheme) || !strings.EqualFold(u.Host, origin.Host) {
			return fallback
		}
		raw = u.RequestURI()
		if raw == "" {
			return fallback
		}
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return fallback
	}
	// Reject escapes that can confuse browsers or split headers.
	if strings.ContainsAny(raw, "\\\x00\r\n\t") || strings.Contains(raw, "://") {
		return fallback
	}
	return raw
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
