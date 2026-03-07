// Package grpcerror centralizes transport-boundary error shaping for game gRPC
// handlers so write-path helpers in different service packages remain aligned.
package grpcerror

import (
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
			return status.Errorf(codes.Internal, "%s: %v", message, err)
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
				return status.Errorf(codes.Internal, "%s: %v", message, err)
			}
		}
	}
	if options.RejectErr == nil {
		options.RejectErr = func(message string) error {
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
		return status.Errorf(codes.Internal, "%s: %v", message, err)
	}
}
