// Package audit contains durable in-product audit writes for game service operations.
//
// This package owns persisted operational audit events that are used for
// security posture, incident analysis, and cross-service debugging.
//
// For distributed tracing, this service still uses package `internal/platform/otel`.
package audit
