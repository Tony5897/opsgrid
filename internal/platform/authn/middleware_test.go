package authn_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Tony5897/opsgrid/internal/platform/authn"
)

type fakeSessionSource struct {
	sess     authn.Session
	getErr   error
	touchErr error
	touchN   int
	lastHash []byte
}

func (f *fakeSessionSource) GetLiveSession(_ context.Context, idHash []byte, _ time.Time) (authn.Session, error) {
	f.lastHash = append([]byte(nil), idHash...)
	if f.getErr != nil {
		return authn.Session{}, f.getErr
	}
	return f.sess, nil
}

func (f *fakeSessionSource) TouchSession(context.Context, []byte, time.Time, time.Duration, time.Duration) error {
	f.touchN++
	return f.touchErr
}

func TestMiddlewareAttachesPrincipal(t *testing.T) {
	t.Parallel()
	raw, hash, err := authn.NewSessionID()
	if err != nil {
		t.Fatal(err)
	}
	uid := uuid.MustParse("01900000-0000-7000-8000-000000000001")
	src := &fakeSessionSource{sess: authn.Session{UserID: uid, IDHash: hash}}
	var got authn.Principal
	var ok bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok = authn.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	mw := authn.Middleware(src, discardLog(), 2*time.Hour, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.AddCookie(&http.Cookie{Name: authn.SessionCookieName, Value: raw})
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent || !ok {
		t.Fatalf("status=%d ok=%v", rr.Code, ok)
	}
	if got.UserID != uid || !bytes.Equal(got.SessionHash, hash) {
		t.Fatalf("principal %+v hash %x", got, got.SessionHash)
	}
	if src.touchN != 1 {
		t.Fatalf("TouchSession calls = %d", src.touchN)
	}
	if !bytes.Equal(src.lastHash, hash) {
		t.Fatal("lookup hash mismatch")
	}
}

func TestMiddlewareClearsCookieOnNotFound(t *testing.T) {
	t.Parallel()
	raw, _, err := authn.NewSessionID()
	if err != nil {
		t.Fatal(err)
	}
	src := &fakeSessionSource{getErr: authn.ErrSessionNotFound}
	var ok bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok = authn.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	mw := authn.Middleware(src, discardLog(), time.Hour, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.AddCookie(&http.Cookie{Name: authn.SessionCookieName, Value: raw})
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)
	if ok {
		t.Fatal("principal must be absent")
	}
	cleared := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == authn.SessionCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("session cookie must be cleared")
	}
}

func TestMiddlewareKeepsCookieOnTransientError(t *testing.T) {
	t.Parallel()
	raw, _, err := authn.NewSessionID()
	if err != nil {
		t.Fatal(err)
	}
	src := &fakeSessionSource{getErr: errors.New("db unavailable")}
	var ok bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok = authn.PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	mw := authn.Middleware(src, discardLog(), time.Hour, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.AddCookie(&http.Cookie{Name: authn.SessionCookieName, Value: raw})
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)
	if ok {
		t.Fatal("principal must be absent")
	}
	for _, c := range rr.Result().Cookies() {
		if c.Name == authn.SessionCookieName && c.MaxAge < 0 {
			t.Fatal("must not clear session cookie on transient errors")
		}
	}
}
