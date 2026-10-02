package webui_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tony5897/opsgrid/internal/platform/webui"
)

// In a source checkout dist/ contains only .gitkeep, so these tests cover the
// placeholder and routing guards; the embedded-build path is covered by the
// E2E suite against the production image.

func TestReservedPrefixesNeverServeSPA(t *testing.T) {
	t.Parallel()
	h := webui.Handler("/v1/", "/auth/")
	for _, p := range []string{"/v1/organizations", "/auth/login"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, p, nil))
		if rr.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", p, rr.Code)
		}
	}
}

func TestRejectsUnsafeMethods(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	webui.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/dispatch", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestPlaceholderWhenNotBuilt(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	webui.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/o/cascade/dispatch", nil))
	if rr.Code != http.StatusServiceUnavailable && rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content-type %q", rr.Header().Get("Content-Type"))
	}
}

func TestCSPIsStrictForScripts(t *testing.T) {
	t.Parallel()
	csp := webui.CSP([]string{"https://files.example"}, []string{"https://files.example"})
	for _, want := range []string{"script-src 'self'", "frame-ancestors 'none'", "object-src 'none'", "connect-src 'self' https://files.example"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP missing %q: %s", want, csp)
		}
	}
	if strings.Contains(csp, "script-src 'self' 'unsafe") {
		t.Errorf("unsafe script source: %s", csp)
	}
}
