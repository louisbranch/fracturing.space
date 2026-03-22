// Package grpcerror maps domain and execution errors into stable gRPC
// status responses.
//
// It is the outermost error boundary in the handler pipeline. Domain
// rejections become codes.FailedPrecondition (with i18n lookup), engine
// errors become codes.Internal (with server-side slog), and structured
// domain codes are preserved when configured.
//
// HandleDomainErrorLocale is the preferred entry point when the caller's
// locale is known — it formats error messages in the appropriate language.
// Callers with a request context should extract the locale via
// grpcmeta.LocaleFromContext before calling.
package grpcerror
