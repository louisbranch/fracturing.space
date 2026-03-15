// Package grpcerror maps domain and execution errors into stable gRPC
// status responses.
//
// It is the outermost error boundary in the handler pipeline. Domain
// rejections become codes.FailedPrecondition (with i18n lookup), engine
// errors become codes.Internal (with server-side slog), and structured
// domain codes are preserved when configured.
//
// grpcerror also provides NormalizeDomainWriteOptions to layer gRPC-aware
// error callbacks onto domainwrite.Options, keeping domainwrite itself
// transport-agnostic.
package grpcerror
