// Package otel provides opt-in OpenTelemetry distributed tracing for
// Fracturing Space services.
//
// Tracing is controlled by two environment variables:
//
//   - FRACTURING_SPACE_OTEL_ENDPOINT — OTLP HTTP endpoint (e.g. http://jaeger:4318).
//     When empty, tracing is disabled and Setup returns a no-op.
//   - FRACTURING_SPACE_OTEL_ENABLED — set to "false" to explicitly disable
//     tracing even when an endpoint is configured.
//
// Call [Setup] early in each service's Run function and defer the returned
// shutdown to flush pending spans on exit.
package otel
