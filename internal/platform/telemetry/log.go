package telemetry

import (
	"context"
	"io"
	"log/slog"
	"regexp"

	"go.opentelemetry.io/otel/trace"
)

// redactedKey matches attribute keys whose values must never be logged:
// session cookies, OIDC tokens, passwords, signed URLs and similar.
var redactedKey = regexp.MustCompile(`(?i)(token|secret|password|passwd|cookie|authorization|signature|signed_?url|api_?key|credential)`)

const redacted = "[REDACTED]"

// NewLogger returns a structured logger that
//   - writes JSON (or text in local development) to w,
//   - adds trace_id/span_id from the active OpenTelemetry span,
//   - adds request-scoped attributes stored with ContextWithAttrs,
//   - redacts sensitive keys at any nesting depth.
func NewLogger(w io.Writer, level slog.Leveler, format string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:       level,
		AddSource:   false,
		ReplaceAttr: redact,
	}
	var h slog.Handler
	if format == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(&contextHandler{Handler: h})
}

func redact(_ []string, a slog.Attr) slog.Attr {
	if redactedKey.MatchString(a.Key) {
		return slog.String(a.Key, redacted)
	}
	return a
}

type attrsKey struct{}

// ContextWithAttrs returns a context carrying attrs, which every log record
// emitted with that context will include (request_id, organization_id,
// actor_user_id, ...). Later calls append to earlier ones.
func ContextWithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing, _ := ctx.Value(attrsKey{}).([]slog.Attr)
	merged := make([]slog.Attr, 0, len(existing)+len(attrs))
	merged = append(merged, existing...)
	merged = append(merged, attrs...)
	return context.WithValue(ctx, attrsKey{}, merged)
}

// AttrsFromContext returns attributes stored by ContextWithAttrs.
func AttrsFromContext(ctx context.Context) []slog.Attr {
	attrs, _ := ctx.Value(attrsKey{}).([]slog.Attr)
	return attrs
}

type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	if attrs := AttrsFromContext(ctx); len(attrs) > 0 {
		r.AddAttrs(attrs...)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}
