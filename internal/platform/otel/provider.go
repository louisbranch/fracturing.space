package otel

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Setup initialises OpenTelemetry tracing for the given service.
//
// Tracing is opt-in: when FRACTURING_SPACE_OTEL_ENDPOINT is empty or
// FRACTURING_SPACE_OTEL_ENABLED is "false", Setup returns a no-op shutdown
// function and no global provider is registered.
//
// The returned shutdown function flushes pending spans and should be deferred
// by the caller.
func Setup(ctx context.Context, serviceName string) (shutdown func(context.Context) error, err error) {
	noop := func(context.Context) error { return nil }

	if strings.EqualFold(os.Getenv("FRACTURING_SPACE_OTEL_ENABLED"), "false") {
		return noop, nil
	}

	endpoint := os.Getenv("FRACTURING_SPACE_OTEL_ENDPOINT")
	if endpoint == "" {
		return noop, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
	)
	if err != nil {
		return noop, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return noop, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp.Shutdown, nil
}
