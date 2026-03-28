package ai

import (
	"context"
	"errors"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// serviceErrorToStatus maps a service-layer error to a gRPC status error.
func serviceErrorToStatus(err error) error {
	if err == nil {
		return nil
	}
	kind := service.ErrorKindOf(err)
	return status.Error(kindToGRPC(kind), err.Error())
}

func kindToGRPC(kind service.ErrorKind) codes.Code {
	switch kind {
	case service.ErrKindInvalidArgument:
		return codes.InvalidArgument
	case service.ErrKindNotFound:
		return codes.NotFound
	case service.ErrKindAlreadyExists:
		return codes.AlreadyExists
	case service.ErrKindPermissionDenied:
		return codes.PermissionDenied
	case service.ErrKindFailedPrecondition:
		return codes.FailedPrecondition
	default:
		return codes.Internal
	}
}

type transportErrorConfig struct {
	Operation string

	DeadlineExceededCode    apperrors.Code
	DeadlineExceededMessage string

	CanceledCode    apperrors.Code
	CanceledMessage string
}

// transportErrorToStatus maps transport-visible failures through one policy
// surface so handlers do not need to re-encode service, app, and generic
// fallback rules independently.
func transportErrorToStatus(err error, cfg transportErrorConfig) error {
	if err == nil {
		return nil
	}

	var svcErr *service.Error
	if errors.As(err, &svcErr) {
		return serviceErrorToStatus(err)
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		return apperrors.HandleError(err, apperrors.DefaultLocale)
	}
	if cfg.DeadlineExceededCode != apperrors.CodeUnknown && errors.Is(err, context.DeadlineExceeded) {
		return apperrors.HandleError(
			apperrors.Wrap(cfg.DeadlineExceededCode, cfg.DeadlineExceededMessage, err),
			apperrors.DefaultLocale,
		)
	}
	if cfg.CanceledCode != apperrors.CodeUnknown && errors.Is(err, context.Canceled) {
		return apperrors.HandleError(
			apperrors.Wrap(cfg.CanceledCode, cfg.CanceledMessage, err),
			apperrors.DefaultLocale,
		)
	}
	return status.Errorf(codes.Internal, "%s: %v", cfg.Operation, err)
}
