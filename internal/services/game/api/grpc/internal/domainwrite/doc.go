// Package domainwrite provides transport-facing write-path helpers for
// command execution and inline projection apply.
//
// It owns the execute → reject/apply → error shaping pipeline. Callers
// supply an Executor (engine) and EventApplier (projection store), then
// configure behavior through Options (require events, error callbacks,
// rejection observers).
//
// The package provides both transport-agnostic helpers (ExecuteAndApply)
// and gRPC-aware entry points (TransportExecuteAndApply) that normalize
// error callbacks, wire audit rejection telemetry, and apply runtime
// configuration.
//
// WritePath bundles the executor, runtime, and optional audit store into
// a single dependency struct that satisfies the Deps interface. Transport
// packages embed or accept WritePath instead of carrying three separate fields.
package domainwrite
