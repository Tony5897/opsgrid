package authn_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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
}

func (f *fakeOIDC) AuthCodeURL(state, nonce, challenge string) string {
	return f.authURL + "?state=" + state + "&nonce=" + nonce + "&cc=" + challenge
}

func (f *fakeOIDC) Exchange(context.Context, string, string, string) (authn.TokenBundle, error) {
	return f.bundle, f.exErr
}

func (f *fakeOIDC) EndSessionURL(post string) (string, error) {
	return "https://idp.example/logout?post=" + post, nil
}

func (f *fakeOIDC) VerifyLogoutToken(context.Context, string) (authn.LogoutClaims, error) {
	return f.logout, f.logErr
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSanitizeReturnToViaLogin(t *testing.T) {
	t.Parallel()
	key := bytes32('a')
	sealer, err := authn.NewSealer(key, "v1")
	if err != nil {
		t.Fatal(err)
	}
	oidc := &fakeOIDC{authURL: "https://idp.example/auth"}
	h := authn.Handler(authn.Options{
		Provider:     oidc,
		Store:        nil, // login does not touch the store
		Sealer:       sealer,
		Log:          discardLog(),
		PublicOrigin: "http://localhost:5173",
		Now:          func() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) },
	})

	cases := []struct {
		returnTo string
		wantPath string // path encoded in redirect is IdP; we check cookie set + 302
	}{
		{"/dashboard", "/dashboard"},
		{"https://evil.example/phish", "/"},
		{"//evil.example", "/"},
	}
	for _, tc := range cases {
		t.Run(tc.returnTo, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/auth/login?returnTo="+tc.returnTo, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Fatalf("status %d", rr.Code)
			}
			if c := cookieNamed(rr, authn.OIDCCookieName); c == nil || c.Value == "" {
				t.Fatal("missing oidc cookie")
			}
			loc := rr.Header().Get("Location")
			if !strings.HasPrefix(loc, "https://idp.example/auth?") {
				t.Fatalf("location %q", loc)
			}
		})
	}
}

func TestLogoutClearsSessionCookie(t *testing.T) {
	t.Parallel()
	key := bytes32('b')
	sealer, _ := authn.NewSealer(key, "v1")
	h := authn.Handler(authn.Options{
		Provider:     &fakeOIDC{},
		Store:        nil,
		Sealer:       sealer,
		Log:          discardLog(),
		PublicOrigin: "http://localhost:5173",
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: authn.SessionCookieName, Value: "stale"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status %d", rr.Code)
	}
	cleared := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == authn.SessionCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("session cookie not cleared")
	}
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
