package grpcerror

import (
	"log/slog"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Internal logs the full error server-side and returns a sanitized gRPC
// Internal status that does not expose implementation details to clients.
func Internal(msg string, err error) error {
	slog.Error(msg, "error", err)
	return status.Error(codes.Internal, msg)
}

// HandleDomainError maps domain errors through the structured app error system
// using the default locale. Prefer HandleDomainErrorLocale when the caller's
// locale is available so error messages are formatted correctly.
//
// TODO: migrate direct HandleDomainError call sites in handler packages
// (adversarytransport, conditiontransport, charactermutationtransport, etc.)
// to HandleDomainErrorLocale so all error paths respect the caller's locale.
func HandleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
}

// HandleDomainErrorLocale maps domain errors through the structured app error
// system using the provided locale string. Callers with a request context
// should extract the locale via grpcmeta.LocaleFromContext before calling.
func HandleDomainErrorLocale(err error, locale string) error {
	return apperrors.HandleError(err, locale)
}

// EnsureStatus guarantees transport boundaries always return gRPC status
// errors, preserving structured domain code mappings when present.
func EnsureStatus(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		return HandleDomainError(err)
	}
	return status.Error(codes.Internal, err.Error())
}

// ApplyErrorWithDomainCodePreserve preserves structured domain errors and wraps
// unknown errors in a gRPC internal status for apply callback flows.
func ApplyErrorWithDomainCodePreserve(message string) func(error) error {
	return func(err error) error {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return err
		}
		return Internal(message, err)
	}
}
