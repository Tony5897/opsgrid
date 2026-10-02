package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/Tony5897/opsgrid/internal/platform/telemetry"
)

func decode(t *testing.T, b *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON log line %q: %v", b.String(), err)
	}
	return m
}

func TestRedactsSensitiveKeys(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := telemetry.NewLogger(&buf, slog.LevelInfo, "json")
	log.Info("login",
		slog.String("session_cookie", "abc"),
		slog.String("refresh_token", "rt"),
		slog.String("Authorization", "Bearer x"),
		slog.String("signed_url", "https://s3/obj?X-Amz-Signature=deadbeef"),
		slog.Group("oidc", slog.String("id_token", "jwt")),
		slog.String("user_id", "u1"),
	)
	out := buf.String()
	for _, secret := range []string{`"abc"`, `"rt"`, "Bearer x", "deadbeef", `"jwt"`} {
		if strings.Contains(out, secret) {
			t.Errorf("secret %s leaked: %s", secret, out)
		}
	}
	if !strings.Contains(out, `"user_id":"u1"`) {
		t.Errorf("non-sensitive attr missing: %s", out)
	}
}

func TestAddsContextAttrsAndTraceIDs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := telemetry.NewLogger(&buf, slog.LevelInfo, "json")

	tp := sdktrace.NewTracerProvider()
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()
	ctx = telemetry.ContextWithAttrs(ctx, slog.String("request_id", "req_1"))
	ctx = telemetry.ContextWithAttrs(ctx, slog.String("organization_id", "org_1"))

	log.InfoContext(ctx, "work order assigned")
	m := decode(t, &buf)

	if m["request_id"] != "req_1" || m["organization_id"] != "org_1" {
		t.Errorf("context attrs missing: %v", m)
	}
	if m["trace_id"] != span.SpanContext().TraceID().String() {
		t.Errorf("trace_id = %v, want %s", m["trace_id"], span.SpanContext().TraceID())
	}
	if m["span_id"] == nil {
		t.Errorf("span_id missing: %v", m)
	}
}

func TestNoTraceIDsWithoutSpan(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	telemetry.NewLogger(&buf, slog.LevelInfo, "json").Info("boot")
	if m := decode(t, &buf); m["trace_id"] != nil {
		t.Fatalf("unexpected trace_id: %v", m)
	}
}
