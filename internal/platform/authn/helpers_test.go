package authn_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/authn"
)

type fakeOIDC struct {
	authURL string
	bundle  authn.TokenBundle
	exErr   error
	logout  authn.LogoutClaims
	logErr  error

	exchangeCalls atomic.Int64
	lastCode      string
	lastVerifier  string
	lastNonce     string
}

func (f *fakeOIDC) Warm(context.Context) error { return nil }

func (f *fakeOIDC) AuthCodeURL(state, nonce, challenge string) string {
	return f.authURL + "?state=" + url.QueryEscape(state) + "&nonce=" + url.QueryEscape(nonce) + "&cc=" + url.QueryEscape(challenge)
}

func (f *fakeOIDC) Exchange(_ context.Context, code, verifier, nonce string) (authn.TokenBundle, error) {
	f.exchangeCalls.Add(1)
	f.lastCode = code
	f.lastVerifier = verifier
	f.lastNonce = nonce
	return f.bundle, f.exErr
}

func (f *fakeOIDC) EndSessionURL(_ context.Context, post string) (string, error) {
	return "https://idp.example/logout?post=" + post, nil
}

func (f *fakeOIDC) VerifyLogoutToken(context.Context, string) (authn.LogoutClaims, error) {
	return f.logout, f.logErr
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func bytes32(b byte) []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = b
	}
	return k
}

func cookieNamed(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// sessionCookieLive returns the last non-cleared session cookie from the response.
func sessionCookieLive(rr *httptest.ResponseRecorder) *http.Cookie {
	var live *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == authn.SessionCookieName && c.Value != "" && c.MaxAge >= 0 {
			live = c
		}
	}
	return live
}

func oidcCookieCleared(rr *httptest.ResponseRecorder) bool {
	for _, c := range rr.Result().Cookies() {
		if c.Name == authn.OIDCCookieName && c.MaxAge < 0 {
			return true
		}
	}
	return false
}

func noSessionCookieSet(rr *httptest.ResponseRecorder) bool {
	return sessionCookieLive(rr) == nil
}

type clock struct {
	t time.Time
}

func (c *clock) Now() time.Time { return c.t }

func doLogin(t *testing.T, h http.Handler, returnTo string) (oidcCookie, state string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/auth/login?returnTo="+url.QueryEscape(returnTo), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("login status %d body %s", rr.Code, rr.Body.String())
	}
	c := cookieNamed(rr, authn.OIDCCookieName)
	if c == nil || c.Value == "" {
		t.Fatal("missing oidc cookie after login")
	}
	loc, err := url.Parse(rr.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state = loc.Query().Get("state")
	if state == "" {
		t.Fatal("missing state in auth redirect")
	}
	return c.Value, state
}

func newTestHandler(t *testing.T, oidc *fakeOIDC, store *authn.Store, sealerKey byte, clk *clock) http.Handler {
	t.Helper()
	sealer, err := authn.NewSealer(bytes32(sealerKey), "v1")
	if err != nil {
		t.Fatal(err)
	}
	if oidc.authURL == "" {
		oidc.authURL = "https://idp.example/auth"
	}
	opts := authn.Options{
		Provider:     oidc,
		Store:        store,
		Sealer:       sealer,
		Log:          discardLog(),
		PublicOrigin: "http://localhost:5173",
		AbsoluteTTL:  12 * time.Hour,
		IdleTTL:      2 * time.Hour,
	}
	if clk != nil {
		opts.Now = clk.Now
	}
	return authn.Handler(opts)
}
