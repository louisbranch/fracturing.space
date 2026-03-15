// Package domainwrite provides transport-facing write-path helpers for
// command execution and inline projection apply.
//
// It owns the execute → reject/apply → error shaping pipeline. Callers
// supply an Executor (engine) and EventApplier (projection store), then
// configure behavior through Options (require events, error callbacks,
// rejection observers).
//
// domainwrite is transport-agnostic — it knows nothing about gRPC codes.
// gRPC-specific error shaping is layered on by grpcerror.NormalizeDomainWriteOptions.
package domainwrite
