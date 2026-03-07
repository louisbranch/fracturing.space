package daggerheart

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type eventApplier interface {
	Apply(context.Context, event.Event) error
}

func (s *DaggerheartService) executeAndApplyDomainCommand(
	ctx context.Context,
	cmd command.Command,
	applier eventApplier,
	options domainwrite.Options,
) (engine.Result, error) {
	normalizeGRPCDefaults(&options)
	result, err := s.stores.WriteRuntime.ExecuteAndApply(ctx, s.stores.Domain, applier, cmd, options)
	if err != nil {
		return result, ensureGRPCStatus(err)
	}
	return result, nil
}

// ensureGRPCStatus wraps plain errors with codes.Internal so callers always
// receive gRPC status errors at the transport boundary. Domain errors
// (apperrors) are converted using their code-to-gRPC mapping.
func ensureGRPCStatus(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		return handleDomainError(err)
	}
	return status.Error(codes.Internal, err.Error())
}

// normalizeGRPCDefaults sets gRPC-status-aware error handlers at the transport
// boundary so the domainwrite package stays transport-agnostic.
func normalizeGRPCDefaults(options *domainwrite.Options) {
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
		options.ApplyErr = domainApplyErrorWithCodePreserve(message)
	}
	if options.RejectErr == nil {
		options.RejectErr = func(message string) error {
			return status.Error(codes.FailedPrecondition, message)
		}
	}
}

// domainApplyErrorWithCodePreserve returns an error handler that preserves
// domain error codes (apperrors) through the apply path instead of flattening
// them to codes.Internal.
func domainApplyErrorWithCodePreserve(message string) func(error) error {
	return func(err error) error {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return err
		}
		return status.Errorf(codes.Internal, "%s: %v", message, err)
	}
}
