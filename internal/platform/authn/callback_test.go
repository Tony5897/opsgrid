package authn_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/authn"
	"github.com/Tony5897/opsgrid/internal/platform/httpx"
)

func TestCallbackRejects(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	t.Run("missing_oidc_cookie", func(t *testing.T) {
		t.Parallel()
		oidc := &fakeOIDC{authURL: "https://idp.example/auth"}
		clk := &clock{t: base}
		h := newTestHandler(t, oidc, nil, 'd', clk)
		req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=x&state=y", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assertCallbackReject(t, rr, http.StatusUnauthorized, apperr.CodeAuthenticationRequired)
		if oidc.exchangeCalls.Load() != 0 {
			t.Fatal("Exchange must not be called")
		}
		if !oidcCookieCleared(rr) {
			t.Fatal("expected oidc cookie cleared")
		}
	})

	t.Run("bad_state", func(t *testing.T) {
		t.Parallel()
		oidc := &fakeOIDC{authURL: "https://idp.example/auth"}
		clk := &clock{t: base}
		h := newTestHandler(t, oidc, nil, 'e', clk)
		cookie, _ := doLogin(t, h, "/dashboard")
		req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test&state=wrong-state", nil)
		req.AddCookie(&http.Cookie{Name: authn.OIDCCookieName, Value: cookie})
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assertCallbackReject(t, rr, http.StatusUnauthorized, apperr.CodeAuthenticationRequired)
		if oidc.exchangeCalls.Load() != 0 {
			t.Fatal("Exchange must not be called")
		}
		if !noSessionCookieSet(rr) {
			t.Fatal("session cookie must not be set")
		}
		if !oidcCookieCleared(rr) {
			t.Fatal("expected oidc cookie cleared")
		}
	})

	t.Run("expired_handshake", func(t *testing.T) {
		t.Parallel()
		oidc := &fakeOIDC{authURL: "https://idp.example/auth"}
		clk := &clock{t: base}
		h := newTestHandler(t, oidc, nil, 'f', clk)
		cookie, state := doLogin(t, h, "/dashboard")
		clk.t = base.Add(11 * time.Minute)
		req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test&state="+state, nil)
		req.AddCookie(&http.Cookie{Name: authn.OIDCCookieName, Value: cookie})
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assertCallbackReject(t, rr, http.StatusUnauthorized, apperr.CodeAuthenticationRequired)
		if oidc.exchangeCalls.Load() != 0 {
			t.Fatal("Exchange must not be called")
		}
		if !noSessionCookieSet(rr) {
			t.Fatal("session cookie must not be set")
		}
	})

	t.Run("missing_code", func(t *testing.T) {
		t.Parallel()
		oidc := &fakeOIDC{authURL: "https://idp.example/auth"}
		clk := &clock{t: base}
		h := newTestHandler(t, oidc, nil, 'g', clk)
		cookie, state := doLogin(t, h, "/dashboard")
		req := httptest.NewRequest(http.MethodGet, "/auth/callback?state="+state, nil)
		req.AddCookie(&http.Cookie{Name: authn.OIDCCookieName, Value: cookie})
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assertCallbackReject(t, rr, http.StatusUnauthorized, apperr.CodeAuthenticationRequired)
		if oidc.exchangeCalls.Load() != 0 {
			t.Fatal("Exchange must not be called")
		}
		if !noSessionCookieSet(rr) {
			t.Fatal("session cookie must not be set")
		}
	})

	t.Run("empty_refresh_token", func(t *testing.T) {
		t.Parallel()
		oidc := &fakeOIDC{
			authURL: "https://idp.example/auth",
			bundle: authn.TokenBundle{
				Issuer: "https://issuer.example", Subject: "sub-1",
				DisplayName: "Test", RefreshToken: "",
			},
		}
		clk := &clock{t: base}
		h := newTestHandler(t, oidc, nil, 'h', clk)
		cookie, state := doLogin(t, h, "/dashboard")
		req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test&state="+state, nil)
		req.AddCookie(&http.Cookie{Name: authn.OIDCCookieName, Value: cookie})
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assertCallbackReject(t, rr, http.StatusServiceUnavailable, apperr.CodeDependencyUnavailable)
		if oidc.exchangeCalls.Load() != 1 {
			t.Fatalf("Exchange calls = %d", oidc.exchangeCalls.Load())
		}
		if !noSessionCookieSet(rr) {
			t.Fatal("session cookie must not be set")
		}
	})
}

func assertCallbackReject(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantCode apperr.Code) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d body %s", rr.Code, wantStatus, rr.Body.String())
	}
	var p httpx.Problem
	if err := json.Unmarshal(rr.Body.Bytes(), &p); err != nil {
		t.Fatalf("problem json: %v body %s", err, rr.Body.String())
	}
	if p.Code != wantCode {
		t.Fatalf("code = %s, want %s", p.Code, wantCode)
	}
}
