// Package telemetry wires OpenTelemetry tracing and metrics and the
// structured slog logger used by every OpsGrid process.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// Options configures Setup.
type Options struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	// OTLPEndpoint (host:port, gRPC). Empty disables exporting; spans are
	// still created so trace IDs appear in logs and responses.
	OTLPEndpoint string
	SampleRatio  float64
	// Insecure disables TLS to the collector (local Compose network).
	Insecure bool
}

// Shutdown flushes and stops telemetry providers.
type Shutdown func(context.Context) error

// Setup installs global tracer and meter providers plus W3C propagators.
func Setup(ctx context.Context, o Options) (Shutdown, error) {
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(o.ServiceName),
		semconv.ServiceVersion(o.ServiceVersion),
		semconv.DeploymentEnvironmentName(o.Environment),
	))
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(o.SampleRatio))
	tpOpts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res), sdktrace.WithSampler(sampler)}
	mpOpts := []sdkmetric.Option{sdkmetric.WithResource(res)}

	if o.OTLPEndpoint != "" {
		traceOpts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(o.OTLPEndpoint)}
		metricOpts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(o.OTLPEndpoint)}
		if o.Insecure {
			traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
			metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
		}
		te, err := otlptracegrpc.New(ctx, traceOpts...)
		if err != nil {
			return nil, fmt.Errorf("otlp trace exporter: %w", err)
		}
		me, err := otlpmetricgrpc.New(ctx, metricOpts...)
		if err != nil {
			return nil, fmt.Errorf("otlp metric exporter: %w", err)
		}
		tpOpts = append(tpOpts, sdktrace.WithBatcher(te))
		mpOpts = append(mpOpts, sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(me, sdkmetric.WithInterval(15*time.Second)),
		))
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)
	mp := sdkmetric.NewMeterProvider(mpOpts...)
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	return func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
	}, nil
}
