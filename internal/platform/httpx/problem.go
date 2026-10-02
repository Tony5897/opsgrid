package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/Tony5897/opsgrid/internal/platform/apperr"
)

// ProblemContentType is the RFC 9457 media type.
const ProblemContentType = "application/problem+json"

// Problem is an RFC 9457 problem document with OpsGrid extension members.
type Problem struct {
	Type      string              `json:"type"`
	Title     string              `json:"title"`
	Status    int                 `json:"status"`
	Detail    string              `json:"detail,omitempty"`
	Instance  string              `json:"instance,omitempty"`
	Code      apperr.Code         `json:"code"`
	RequestID string              `json:"requestId,omitempty"`
	TraceID   string              `json:"traceId,omitempty"`
	Details   map[string]any      `json:"details,omitempty"`
	Errors    []apperr.FieldError `json:"errors,omitempty"`
}

type codeInfo struct {
	status int
	title  string
}

// codes maps each stable error code to its HTTP status and generic title.
// Every apperr.Code must appear here (enforced by a unit test).
var codes = map[apperr.Code]codeInfo{
	apperr.CodeAuthenticationRequired: {http.StatusUnauthorized, "Authentication required"},
	apperr.CodeForbidden:              {http.StatusForbidden, "Forbidden"},
	apperr.CodeTenantNotFound:         {http.StatusNotFound, "Organization not found"},
	apperr.CodeResourceNotFound:       {http.StatusNotFound, "Resource not found"},
	apperr.CodeValidationFailed:       {http.StatusUnprocessableEntity, "Validation failed"},
	apperr.CodePreconditionRequired:   {http.StatusPreconditionRequired, "Precondition required"},
	apperr.CodeVersionConflict:        {http.StatusPreconditionFailed, "Version conflict"},
	apperr.CodeScheduleConflict:       {http.StatusConflict, "Schedule conflict"},
	apperr.CodeInvalidTransition:      {http.StatusConflict, "Invalid state transition"},
	apperr.CodeIdempotencyConflict:    {http.StatusUnprocessableEntity, "Idempotency key reused"},
	apperr.CodeIdempotencyKeyRequired: {http.StatusBadRequest, "Idempotency key required"},
	apperr.CodeRequestTooLarge:        {http.StatusRequestEntityTooLarge, "Request too large"},
	apperr.CodeUnsupportedMediaType:   {http.StatusUnsupportedMediaType, "Unsupported media type"},
	apperr.CodeRateLimited:            {http.StatusTooManyRequests, "Too many requests"},
	apperr.CodeDependencyUnavailable:  {http.StatusServiceUnavailable, "Service temporarily unavailable"},
	apperr.CodeInternal:               {http.StatusInternalServerError, "Internal server error"},
}

// StatusOf returns the HTTP status for code.
func StatusOf(code apperr.Code) int {
	if ci, ok := codes[code]; ok {
		return ci.status
	}
	return http.StatusInternalServerError
}

// problemType returns a stable URI identifying the error type.
func problemType(code apperr.Code) string {
	return "https://opsgrid.dev/problems/" + string(code)
}

// ProblemFromError converts any error into a client-safe Problem. Untyped
// errors become INTERNAL_ERROR with no detail, so database or library error
// text can never reach the client (ADR-012).
func ProblemFromError(ctx context.Context, err error) Problem {
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		ae = apperr.New(apperr.CodeInternal, "")
	}
	ci, ok := codes[ae.Code]
	if !ok {
		ci = codes[apperr.CodeInternal]
		ae = apperr.New(apperr.CodeInternal, "")
	}
	p := Problem{
		Type:      problemType(ae.Code),
		Title:     ci.title,
		Status:    ci.status,
		Detail:    ae.Message,
		Code:      ae.Code,
		RequestID: RequestIDFromContext(ctx),
		Details:   ae.Details,
		Errors:    ae.Fields,
	}
	if ae.Code == apperr.CodeInternal {
		p.Detail = "An unexpected error occurred. Quote the request ID when reporting it."
		p.Details = nil
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		p.TraceID = sc.TraceID().String()
	}
	return p
}

// WriteError logs err at the appropriate level and writes the problem
// response. Server-side failures are logged with their full cause; client
// errors are logged at debug to avoid noise.
func WriteError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	p := ProblemFromError(r.Context(), err)
	p.Instance = r.URL.Path
	if p.Status >= 500 {
		log.ErrorContext(r.Context(), "request failed", slog.String("code", string(p.Code)), slog.Any("error", err))
	} else {
		log.DebugContext(r.Context(), "request rejected", slog.String("code", string(p.Code)), slog.Any("error", err))
	}
	WriteProblem(w, p)
}

// WriteProblem serializes p with the problem+json media type.
func WriteProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", ProblemContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}
