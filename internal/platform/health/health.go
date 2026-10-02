// Package health implements liveness and readiness endpoints.
//
//   - /livez  : the process is running and can serve HTTP. Never checks
//     dependencies (a DB outage must not cause restarts).
//   - /readyz : the process should receive traffic. Fails while shutting down
//     or when a critical dependency check fails. Non-critical failures (Redis
//     for the API, per the failure matrix) report "degraded" but stay ready.
package health

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/httpx"
)

// Check is one dependency probe.
type Check struct {
	Name     string
	Critical bool
	Fn       func(context.Context) error
}

// Health aggregates checks and the readiness flag.
type Health struct {
	checks  []Check
	timeout time.Duration
	ready   atomic.Bool
}

// New returns a Health that starts ready.
func New(timeout time.Duration, checks ...Check) *Health {
	h := &Health{checks: checks, timeout: timeout}
	h.ready.Store(true)
	return h
}

// SetReady toggles readiness (false during graceful shutdown).
func (h *Health) SetReady(ready bool) { h.ready.Store(ready) }

// Status values.
const (
	StatusOK           = "ok"
	StatusDegraded     = "degraded"
	StatusUnavailable  = "unavailable"
	StatusShuttingDown = "shutting_down"
)

// Report is the readiness response body. Error strings are deliberately
// omitted: these endpoints live on the internal port, but they still never
// expose connection details.
type Report struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Livez always returns 200 while the process can serve HTTP.
func (h *Health) Livez(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, Report{Status: StatusOK})
}

// Readyz runs all checks concurrently under one timeout.
func (h *Health) Readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !h.ready.Load() {
		httpx.WriteJSON(w, http.StatusServiceUnavailable, Report{Status: StatusShuttingDown})
		return
	}
	rep := h.Run(r.Context())
	status := http.StatusOK
	if rep.Status == StatusUnavailable {
		status = http.StatusServiceUnavailable
	}
	httpx.WriteJSON(w, status, rep)
}

// Run executes the checks and aggregates the result.
func (h *Health) Run(ctx context.Context) Report {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	type result struct {
		check Check
		err   error
	}
	results := make([]result, len(h.checks))
	var wg sync.WaitGroup
	for i, c := range h.checks {
		wg.Go(func() {
			results[i] = result{check: c, err: c.Fn(ctx)}
		})
	}
	wg.Wait()

	rep := Report{Status: StatusOK, Checks: make(map[string]string, len(results))}
	for _, res := range results {
		if res.err == nil {
			rep.Checks[res.check.Name] = StatusOK
			continue
		}
		rep.Checks[res.check.Name] = "failing"
		if res.check.Critical {
			rep.Status = StatusUnavailable
		} else if rep.Status == StatusOK {
			rep.Status = StatusDegraded
		}
	}
	return rep
}

// Mux returns a handler serving /livez and /readyz.
func (h *Health) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", h.Livez)
	mux.HandleFunc("GET /readyz", h.Readyz)
	return mux
}
