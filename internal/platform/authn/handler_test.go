package authn_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/authn"
)

func TestSanitizeReturnToViaLogin(t *testing.T) {
	t.Parallel()
	clk := &clock{t: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)}
	h := newTestHandler(t, &fakeOIDC{authURL: "https://idp.example/auth"}, nil, 'a', clk)

	cases := []struct {
		returnTo string
	}{
		{"/dashboard"},
		{"https://evil.example/phish"},
		{"//evil.example"},
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
	h := newTestHandler(t, &fakeOIDC{}, nil, 'b', nil)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
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

func TestGetLogoutRejected(t *testing.T) {
	t.Parallel()
	h := newTestHandler(t, &fakeOIDC{}, nil, 'c', nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/auth/logout", nil))
	if rr.Code != http.StatusMethodNotAllowed && rr.Code != http.StatusNotFound {
		t.Fatalf("GET logout must not succeed, got %d", rr.Code)
	}
}
