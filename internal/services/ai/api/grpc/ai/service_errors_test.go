package ai

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestTransportErrorToStatus(t *testing.T) {
	t.Parallel()

	cfg := transportErrorConfig{
		Operation:               "run campaign turn",
		DeadlineExceededCode:    apperrors.CodeAIOrchestrationTimedOut,
		DeadlineExceededMessage: "campaign turn timed out",
		CanceledCode:            apperrors.CodeAIOrchestrationCanceled,
		CanceledMessage:         "campaign turn canceled",
	}

	t.Run("service error", func(t *testing.T) {
		t.Parallel()

		err := transportErrorToStatus(service.Errorf(service.ErrKindInvalidArgument, "bad input"), cfg)
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
	})

	t.Run("app error", func(t *testing.T) {
		t.Parallel()

		err := transportErrorToStatus(
			apperrors.Wrap(apperrors.CodeAIOrchestrationStepLimitExceeded, "step limit exceeded", errors.New("boom")),
			cfg,
		)
		grpcassert.StatusCode(t, err, codes.Internal)
		assertStatusReason(t, err, apperrors.CodeAIOrchestrationStepLimitExceeded)
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		t.Parallel()

		err := transportErrorToStatus(context.DeadlineExceeded, cfg)
		grpcassert.StatusCode(t, err, codes.DeadlineExceeded)
		assertStatusReason(t, err, apperrors.CodeAIOrchestrationTimedOut)
	})

	t.Run("canceled", func(t *testing.T) {
		t.Parallel()

		err := transportErrorToStatus(context.Canceled, cfg)
		grpcassert.StatusCode(t, err, codes.Canceled)
		assertStatusReason(t, err, apperrors.CodeAIOrchestrationCanceled)
	})

	t.Run("generic fallback", func(t *testing.T) {
		t.Parallel()

		err := transportErrorToStatus(errors.New("boom"), transportErrorConfig{Operation: "search system reference"})
		grpcassert.StatusCode(t, err, codes.Internal)
	})
}
