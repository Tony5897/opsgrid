package telemetry_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"

	"github.com/Tony5897/opsgrid/internal/platform/telemetry"
)

// Setup must succeed without a collector. This also guards against semconv
// schema-URL drift between our resource and the SDK's default resource,
// which otherwise only fails at process start.
func TestSetupWithoutExporter(t *testing.T) {
	shutdown, err := telemetry.Setup(context.Background(), telemetry.Options{
		ServiceName: "opsgrid-test", ServiceVersion: "test", Environment: "test", SampleRatio: 1,
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	_, span := otel.Tracer("t").Start(context.Background(), "op")
	if !span.SpanContext().IsValid() {
		t.Error("spans must be recorded even without an exporter (trace IDs in logs)")
	}
	span.End()
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
