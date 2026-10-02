package httpx_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/httpx"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestEveryCodeHasAStatus(t *testing.T) {
	t.Parallel()
	for _, c := range apperr.AllCodes() {
		p := httpx.ProblemFromError(t.Context(), apperr.New(c, "x"))
		if p.Code != c {
			t.Errorf("code %s mapped to %s (missing from status table?)", c, p.Code)
		}
		if p.Status < 400 {
			t.Errorf("code %s has non-error status %d", c, p.Status)
		}
	}
}

func TestUntypedErrorNeverLeaksText(t *testing.T) {
	t.Parallel()
	leak := `ERROR: duplicate key value violates unique constraint "work_orders_pkey" (SQLSTATE 23505)`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	httpx.WriteError(rr, req, discard, errors.New(leak))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != httpx.ProblemContentType {
		t.Fatalf("content-type = %q", ct)
	}
	if body := rr.Body.String(); strings.Contains(body, "SQLSTATE") || strings.Contains(body, "work_orders_pkey") {
		t.Fatalf("internal error text leaked: %s", body)
	}
}

func TestInternalTypedErrorAlsoHidesMessage(t *testing.T) {
	t.Parallel()
	p := httpx.ProblemFromError(t.Context(), apperr.Wrap(errors.New("x"), apperr.CodeInternal, "pgx: conn busy"))
	if strings.Contains(p.Detail, "pgx") {
		t.Fatalf("internal detail leaked: %q", p.Detail)
	}
}

func TestProblemIncludesRequestID(t *testing.T) {
	t.Parallel()
	h := httpx.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteError(w, r, discard, apperr.NotFound("work order"))
	}), httpx.RequestID(false))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/wo/1", nil))

	var p httpx.Problem
	if err := json.Unmarshal(rr.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Status != 404 || p.Code != apperr.CodeResourceNotFound {
		t.Fatalf("unexpected problem: %+v", p)
	}
	if p.RequestID == "" || p.RequestID != rr.Header().Get(httpx.RequestIDHeader) {
		t.Fatalf("request id mismatch: body=%q header=%q", p.RequestID, rr.Header().Get(httpx.RequestIDHeader))
	}
}

func TestRequestIDInboundTrust(t *testing.T) {
	t.Parallel()
	echo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, httpx.RequestIDFromContext(r.Context()))
	})
	cases := []struct {
		name    string
		trust   bool
		inbound string
		keep    bool
	}{
		{"untrusted ignored", false, "req_abcdefgh", false},
		{"trusted valid kept", true, "req_abcdefgh", true},
		{"trusted injection rejected", true, "x\ninjected=1 level=ERROR", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(httpx.RequestIDHeader, tc.inbound)
			rr := httptest.NewRecorder()
			httpx.RequestID(tc.trust)(echo).ServeHTTP(rr, req)
			got := rr.Body.String()
			if (got == tc.inbound) != tc.keep {
				t.Fatalf("id = %q, keep=%v", got, tc.keep)
			}
			if !strings.HasPrefix(got, "req_") {
				t.Fatalf("bad id %q", got)
			}
		})
	}
}

func TestRecoverReturnsProblem(t *testing.T) {
	t.Parallel()
	h := httpx.Recover(discard)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != 500 || strings.Contains(rr.Body.String(), "boom") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	httpx.SecurityHeaders(true, "")(http.NotFoundHandler()).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "Strict-Transport-Security", "Cross-Origin-Opener-Policy"} {
		if rr.Header().Get(h) == "" {
			t.Errorf("missing %s", h)
		}
	}
}

func TestCSRF(t *testing.T) {
	t.Parallel()
	mw, err := httpx.CSRF(discard, "http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	ok := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	cases := []struct {
		name    string
		method  string
		headers map[string]string
		want    int
	}{
		{"safe method passes", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site"}, 204},
		{"same-origin with header", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin", httpx.CSRFHeader: "1"}, 204},
		{"same-origin missing header", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin"}, 403},
		{"cross-site rejected even with header", http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site", httpx.CSRFHeader: "1"}, 403},
		{"trusted dev origin", http.MethodPatch, map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "http://localhost:5173", httpx.CSRFHeader: "1"}, 204},
		{"untrusted origin", http.MethodDelete, map[string]string{"Origin": "https://evil.example", httpx.CSRFHeader: "1"}, 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, "http://api.local/v1/x", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rr := httptest.NewRecorder()
			ok.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status = %d, want %d (body %s)", rr.Code, tc.want, rr.Body.String())
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	t.Parallel()
	type in struct {
		Title string `json:"title"`
	}
	cases := []struct {
		name, ct, body string
		limit          int64
		want           apperr.Code
	}{
		{"ok", "application/json", `{"title":"x"}`, 1024, ""},
		{"unknown field (mass assignment)", "application/json", `{"title":"x","organizationId":"y"}`, 1024, apperr.CodeValidationFailed},
		{"trailing data", "application/json", `{"title":"x"}{"title":"y"}`, 1024, apperr.CodeValidationFailed},
		{"malformed", "application/json", `{"title":`, 1024, apperr.CodeValidationFailed},
		{"wrong type", "application/json", `{"title":5}`, 1024, apperr.CodeValidationFailed},
		{"empty", "application/json", ``, 1024, apperr.CodeValidationFailed},
		{"wrong content type", "text/plain", `{"title":"x"}`, 1024, apperr.CodeUnsupportedMediaType},
		{"too large", "application/json", `{"title":"` + strings.Repeat("a", 100) + `"}`, 16, apperr.CodeRequestTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.ct)
			rr := httptest.NewRecorder()
			req.Body = http.MaxBytesReader(rr, req.Body, tc.limit)
			var dst in
			err := httpx.DecodeJSON(req, &dst)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if got := apperr.CodeOf(err); got != tc.want {
				t.Fatalf("code = %s, want %s (err %v)", got, tc.want, err)
			}
		})
	}
}
