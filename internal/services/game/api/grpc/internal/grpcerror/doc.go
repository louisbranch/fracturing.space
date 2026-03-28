// Package grpcerror maps domain and execution errors into stable gRPC
// status responses.
//
// It is the outermost error boundary in the handler pipeline. Domain
// rejections become codes.FailedPrecondition (with i18n lookup), engine
// errors become codes.Internal (with server-side slog), and structured
// domain codes are preserved when configured.
//
// HandleDomainErrorContext is the preferred entry point when a request context
// is available. HandleDomainErrorLocale remains available for non-request
// seams that already resolved locale explicitly.
package grpcerror
