package otelx

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Setup installs a global TracerProvider when endpoint is non-empty.
// Returns a shutdown func (no-op when disabled).
func Setup(ctx context.Context, serviceName, endpoint string, log *slog.Logger) (func(context.Context) error, error) {
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}
	if log == nil {
		log = slog.Default()
	}
	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	)
	otel.SetTracerProvider(tp)
	log.Info("otel tracing enabled", "endpoint", endpoint, "service", serviceName)
	return tp.Shutdown, nil
}

// Tracer returns the named tracer (noop if not configured).
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// SIPAttrs builds common SIP span attributes.
func SIPAttrs(method, callID, tenant, route string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("sip.method", method),
		attribute.String("sip.call_id", callID),
		attribute.String("sip.tenant", tenant),
		attribute.String("sip.route", route),
	}
}
