// Package interceptors provides gRPC middleware for game services.
//
// Interceptor ordering (outermost to innermost):
//
//  1. metadata — establishes RequestID and InvocationID in context.
//     Must be first so all downstream interceptors and handlers have IDs.
//
//  2. internal_identity — validates internal service identity headers.
//     Runs before session lock to reject unauthorized callers early.
//
//  3. telemetry — emits audit events with gRPC status codes.
//     Wraps handler execution to observe the final outcome including
//     error conversion results.
//
//  4. session_lock — blocks mutators during active sessions.
//     Cheap gate check; rejects before handler execution.
//
//  5. error_conversion — converts domain errors to gRPC status codes.
//     Innermost so it translates handler errors before telemetry observes them.
//
// Verify ordering in app/bootstrap_transport.go where interceptors are chained.
package interceptors
