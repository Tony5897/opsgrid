// Package httpx contains the HTTP plumbing shared by OpsGrid processes: the
// middleware pipeline, RFC 9457 problem responses, strict JSON decoding and
// server construction. It holds no business logic.
package httpx

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
	"github.com/Tony5897/opsgrid/internal/platform/telemetry"
)

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so that the first argument is the outermost.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// ---- Request ID -----------------------------------------------------------

// RequestIDHeader is echoed on every response.
const RequestIDHeader = "X-Request-Id"

type requestIDKey struct{}

// validInboundID accepts a caller-supplied ID only if it is short and
// harmless, so it cannot be used for log injection.
var validInboundID = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// RequestID assigns each request a ULID-based ID (or accepts a well-formed
// inbound one from a trusted proxy), stores it in the context and logs, and
// echoes it in the response.
func RequestID(trustInbound bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if !trustInbound || !validInboundID.MatchString(id) {
				id = NewRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey{}, id)
			ctx = telemetry.ContextWithAttrs(ctx, slog.String("request_id", id))
			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// NewRequestID returns a new time-ordered request identifier.
func NewRequestID() string {
	return "req_" + ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// RequestIDFromContext returns the request ID, or "" outside a request.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// ---- Recover --------------------------------------------------------------

// Recover converts panics into 500 problem responses and logs the stack.
// http.ErrAbortHandler is re-panicked, as net/http expects.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() { //nolint:contextcheck // uses the request context via r.Context()
				rec := recover()
				if rec == nil {
					return
				}
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}
				log.ErrorContext(r.Context(), "panic recovered",
					slog.Any("panic", rec), slog.String("stack", string(debug.Stack())))
				WriteError(w, r, log, apperr.Wrap(fmt.Errorf("panic: %v", rec), apperr.CodeInternal, ""))
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// ---- Access log -----------------------------------------------------------

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += int64(n)
	return n, err
}

// Unwrap lets http.ResponseController reach Flush/Hijack on the underlying
// writer (needed for WebSocket upgrades).
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// AccessLog emits one structured line per request. It never logs query
// strings (they may contain signed-URL parameters) or headers.
func AccessLog(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			level := slog.LevelInfo
			switch {
			case status >= 500:
				level = slog.LevelError
			case status >= 400:
				level = slog.LevelWarn
			}
			log.LogAttrs(r.Context(), level, "http request",
				slog.String("method", r.Method),
				slog.String("route", routeOf(r)),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int64("bytes", rec.bytes),
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			)
		})
	}
}

// routeOf returns the matched ServeMux pattern (low-cardinality) when known.
func routeOf(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return "unmatched"
}

// ---- Security headers ------------------------------------------------------

// SecurityHeaders sets baseline response headers. csp may be empty for
// API-only handlers (a restrictive default is used).
func SecurityHeaders(production bool, csp string) Middleware {
	if csp == "" {
		csp = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy", csp)
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")
			h.Set("Permissions-Policy", "camera=(self), geolocation=(), microphone=(), payment=(), usb=()")
			h.Set("X-Frame-Options", "DENY")
			if production {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ---- Body limit -----------------------------------------------------------

// MaxBytes caps request bodies; DecodeJSON reports overflow as 413.
func MaxBytes(n int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

// ---- CSRF (ADR-019) ---------------------------------------------------------

// CSRFHeader must be present on unsafe requests. Cross-site HTML forms cannot
// set custom headers, and the API never grants a CORS preflight.
const CSRFHeader = "X-OpsGrid-CSRF"

// CSRF combines Go's Fetch-Metadata based CrossOriginProtection with a
// required custom header on unsafe methods. trustedOrigins lists origins
// (e.g. the Vite dev server) allowed to make cross-origin unsafe requests.
func CSRF(log *slog.Logger, trustedOrigins ...string) (Middleware, error) {
	cop := http.NewCrossOriginProtection()
	for _, o := range trustedOrigins {
		if err := cop.AddTrustedOrigin(o); err != nil {
			return nil, fmt.Errorf("csrf trusted origin %q: %w", o, err)
		}
	}
	reject := func(w http.ResponseWriter, r *http.Request, reason string) {
		log.WarnContext(r.Context(), "csrf rejected", slog.String("reason", reason))
		WriteError(w, r, log, apperr.New(apperr.CodeForbidden, "Cross-site request rejected."))
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			if err := cop.Check(r); err != nil {
				reject(w, r, "cross-origin")
				return
			}
			if r.Header.Get(CSRFHeader) != "1" {
				reject(w, r, "missing csrf header")
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}
