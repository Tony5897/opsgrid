//go:build integration

package authn_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/authn"
	"github.com/Tony5897/opsgrid/internal/platform/database/dbtest"
)

func TestCallbackCreatesSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.New(t).Open(t, "opsgrid_app", 4)
	store := authn.NewStore(db)
	clk := &clock{t: time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)}
	oidc := &fakeOIDC{
		authURL: "https://idp.example/auth",
		bundle: authn.TokenBundle{
			Issuer:       "http://localhost:8180/realms/opsgrid",
			Subject:      "callback-success-sub",
			Email:        "owen@cascade.example",
			DisplayName:  "Owen Park",
			RefreshToken: "refresh-token-value",
		},
	}
	h := newTestHandler(t, oidc, store, 'i', clk)
	cookie, state := doLogin(t, h, "/dashboard")

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=authcode&state="+state, nil)
	req.AddCookie(&http.Cookie{Name: authn.OIDCCookieName, Value: cookie})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/dashboard" {
		t.Fatalf("Location = %q", loc)
	}
	if oidc.exchangeCalls.Load() != 1 {
		t.Fatalf("Exchange calls = %d", oidc.exchangeCalls.Load())
	}
	live := sessionCookieLive(rr)
	if live == nil || live.Value == "" {
		t.Fatal("missing session cookie")
	}
	if !live.Secure || !live.HttpOnly || live.SameSite != http.SameSiteLaxMode || live.Path != "/" {
		t.Fatalf("cookie flags: secure=%v httpOnly=%v sameSite=%v path=%q", live.Secure, live.HttpOnly, live.SameSite, live.Path)
	}
	sess, err := store.GetLiveSession(ctx, authn.HashSessionID(live.Value), clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if sess.UserID.String() == "" {
		t.Fatal("empty user id")
	}
}

func TestCallbackRotatesPriorSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.New(t).Open(t, "opsgrid_app", 4)
	store := authn.NewStore(db)
	sealer, err := authn.NewSealer(bytes32('j'), "v1")
	if err != nil {
		t.Fatal(err)
	}
	clk := &clock{t: time.Date(2026, 10, 14, 16, 0, 0, 0, time.UTC)}

	userID, err := store.UpsertUser(ctx, authn.UserUpsert{
		Issuer: "http://localhost:8180/realms/opsgrid", Subject: "fixation-sub",
		DisplayName: "Fix",
	})
	if err != nil {
		t.Fatal(err)
	}
	oldRaw, oldHash, err := authn.NewSessionID()
	if err != nil {
		t.Fatal(err)
	}
	enc, err := sealer.SealBinary([]byte("old-refresh"))
	if err != nil {
		t.Fatal(err)
	}
	now := clk.Now()
	if err := store.CreateSession(ctx, authn.CreateSessionParams{
		UserID: userID, IDHash: oldHash,
		AbsoluteExpiresAt: now.Add(12 * time.Hour), IdleExpiresAt: now.Add(2 * time.Hour),
		RefreshTokenEnc: enc, RefreshTokenKeyID: "v1",
	}); err != nil {
		t.Fatal(err)
	}

	oidc := &fakeOIDC{
		authURL: "https://idp.example/auth",
		bundle: authn.TokenBundle{
			Issuer: "http://localhost:8180/realms/opsgrid", Subject: "fixation-sub",
			DisplayName: "Fix", RefreshToken: "new-refresh",
		},
	}
	h := newTestHandler(t, oidc, store, 'j', clk)
	oidcCookie, state := doLogin(t, h, "/dashboard")

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=authcode&state="+state, nil)
	req.AddCookie(&http.Cookie{Name: authn.OIDCCookieName, Value: oidcCookie})
	req.AddCookie(&http.Cookie{Name: authn.SessionCookieName, Value: oldRaw})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}

	newLive := sessionCookieLive(rr)
	if newLive == nil || newLive.Value == "" {
		t.Fatal("missing new session cookie")
	}
	if newLive.Value == oldRaw {
		t.Fatal("session id must rotate")
	}
	if _, err := store.GetLiveSession(ctx, oldHash, clk.Now()); !errors.Is(err, authn.ErrSessionNotFound) {
		t.Fatalf("old session should be gone: %v", err)
	}
	if _, err := store.GetLiveSession(ctx, authn.HashSessionID(newLive.Value), clk.Now()); err != nil {
		t.Fatalf("new session: %v", err)
	}
}
