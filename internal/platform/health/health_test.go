package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/health"
)

func ok(context.Context) error   { return nil }
func fail(context.Context) error { return errors.New("dial tcp 10.0.0.5:5432: secret-host") }

func get(t *testing.T, h *health.Health, path string) (int, health.Report) {
	t.Helper()
	rr := httptest.NewRecorder()
	h.Mux().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
	var rep health.Report
	if err := json.Unmarshal(rr.Body.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	return rr.Code, rep
}

func TestReadiness(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		checks     []health.Check
		wantCode   int
		wantStatus string
	}{
		{"all ok", []health.Check{{"postgres", true, ok}, {"redis", false, ok}}, 200, health.StatusOK},
		{"non-critical failing is degraded but ready", []health.Check{{"postgres", true, ok}, {"redis", false, fail}}, 200, health.StatusDegraded},
		{"critical failing is unavailable", []health.Check{{"postgres", true, fail}, {"redis", false, ok}}, 503, health.StatusUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, rep := get(t, health.New(time.Second, tc.checks...), "/readyz")
			if code != tc.wantCode || rep.Status != tc.wantStatus {
				t.Fatalf("got %d/%s, want %d/%s", code, rep.Status, tc.wantCode, tc.wantStatus)
			}
		})
	}
}

func TestShuttingDownIsNotReadyButLive(t *testing.T) {
	t.Parallel()
	h := health.New(time.Second, health.Check{Name: "postgres", Critical: true, Fn: ok})
	h.SetReady(false)
	if code, rep := get(t, h, "/readyz"); code != 503 || rep.Status != health.StatusShuttingDown {
		t.Fatalf("readyz = %d/%s", code, rep.Status)
	}
	if code, _ := get(t, h, "/livez"); code != 200 {
		t.Fatalf("livez = %d", code)
	}
}

func TestLivezNeverRunsChecks(t *testing.T) {
	t.Parallel()
	h := health.New(time.Second, health.Check{Name: "postgres", Critical: true, Fn: func(context.Context) error {
		t.Error("livez must not call dependency checks")
		return nil
	}})
	if code, _ := get(t, h, "/livez"); code != 200 {
		t.Fatalf("livez = %d", code)
	}
}

func TestCheckTimeoutIsBounded(t *testing.T) {
	t.Parallel()
	slow := func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	h := health.New(50*time.Millisecond, health.Check{Name: "postgres", Critical: true, Fn: slow})
	start := time.Now()
	code, _ := get(t, h, "/readyz")
	if code != 503 || time.Since(start) > time.Second {
		t.Fatalf("code=%d elapsed=%s", code, time.Since(start))
	}
}

func TestErrorDetailsNotExposed(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	health.New(time.Second, health.Check{Name: "postgres", Critical: true, Fn: fail}).
		Mux().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if body := rr.Body.String(); strings.Contains(body, "secret-host") || strings.Contains(body, "10.0.0.5") {
		t.Fatalf("error detail leaked: %s", body)
	}
}
