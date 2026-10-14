package authn

import (
	"context"
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
	AuthCodeURL(state, nonce, challenge string) string
	Exchange(ctx context.Context, code, verifier, nonce string) (TokenBundle, error)
	EndSessionURL(postLogoutRedirect string) (string, error)
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
	mux.HandleFunc("POST /auth/logout", h.logout)
	mux.HandleFunc("GET /auth/logout", h.logout)
	mux.HandleFunc("POST /auth/backchannel-logout", h.backchannel)
	return mux
}

type handlers struct {
	opts Options
}

func (h *handlers) login(w http.ResponseWriter, r *http.Request) {
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
	if r.URL.Query().Get("state") != hs.State {
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
	// Session fixation: always issue a fresh id; any prior cookie is discarded.
	ClearSessionCookie(w)
	SetSessionCookie(w, rawID, h.opts.AbsoluteTTL)
	http.Redirect(w, r, hs.ReturnTo, http.StatusFound)
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
	endURL, err := h.opts.Provider.EndSessionURL(postLogout)
	if err != nil {
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

// sanitizeReturnTo allows only same-origin relative paths (open-redirect safe).
func sanitizeReturnTo(raw, publicOrigin string) string {
	fallback := "/"
	if raw == "" {
		return fallback
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fallback
	}
	origin, err := url.Parse(publicOrigin)
	if err != nil {
		return fallback
	}
	if !strings.EqualFold(u.Scheme, origin.Scheme) || !strings.EqualFold(u.Host, origin.Host) {
		return fallback
	}
	path := u.RequestURI()
	if path == "" {
		return fallback
	}
	return path
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
