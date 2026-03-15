package grpcerror

import (
	"log/slog"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	errori18n "github.com/louisbranch/fracturing.space/internal/platform/errors/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Internal logs the full error server-side and returns a sanitized gRPC
// Internal status that does not expose implementation details to clients.
func Internal(msg string, err error) error {
	slog.Error(msg, "error", err)
	return status.Error(codes.Internal, msg)
}

// NormalizeDomainWriteOptionsConfig controls default gRPC mapping behavior for
// domain write helper options.
type NormalizeDomainWriteOptionsConfig struct {
	// PreserveDomainCodeOnApply keeps structured domain errors intact in apply
	// callbacks instead of flattening them to codes.Internal.
	PreserveDomainCodeOnApply bool
}

// HandleDomainError maps domain errors through the structured app error system.
func HandleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
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

// NormalizeDomainWriteOptions applies gRPC-aware defaults to domainwrite
// options while allowing callers to override callbacks explicitly.
func NormalizeDomainWriteOptions(options *domainwrite.Options, config NormalizeDomainWriteOptionsConfig) {
	if options == nil {
		return
	}
	if options.ExecuteErr == nil {
		message := options.ExecuteErrMessage
		if message == "" {
			message = "execute domain command"
		}
		options.ExecuteErr = func(err error) error {
			if engine.IsNonRetryable(err) {
				return status.Errorf(codes.FailedPrecondition, "%s: %v", message, err)
			}
			return Internal(message, err)
		}
	}
	if options.ApplyErr == nil {
		message := options.ApplyErrMessage
		if message == "" {
			message = "apply event"
		}
		if config.PreserveDomainCodeOnApply {
			options.ApplyErr = ApplyErrorWithDomainCodePreserve(message)
		} else {
			options.ApplyErr = func(err error) error {
				return Internal(message, err)
			}
		}
	}
	if options.RejectErr == nil {
		options.RejectErr = func(code, message string) error {
			cat := errori18n.GetCatalog("en-US")
			if localized := cat.Format(code, nil); localized != code {
				return status.Error(codes.FailedPrecondition, localized)
			}
			return status.Error(codes.FailedPrecondition, message)
		}
	}
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
