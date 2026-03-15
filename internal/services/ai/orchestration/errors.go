package orchestration

import (
	"context"
	"errors"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
)

var (
	// ErrStepLimit indicates the model exceeded the allowed tool loop depth.
	ErrStepLimit = apperrors.New(
		apperrors.CodeAIOrchestrationStepLimitExceeded,
		"campaign orchestration exceeded tool loop limit",
	)

	// ErrNarrationNotCommitted indicates the model returned narration without an
	// authoritative GM output commit.
	ErrNarrationNotCommitted = apperrors.New(
		apperrors.CodeAIOrchestrationNarrationNotCommitted,
		"campaign orchestration did not commit gm output",
	)

	// ErrEmptyOutput indicates the orchestration surface returned no final
	// narration.
	ErrEmptyOutput = apperrors.New(
		apperrors.CodeAIOrchestrationEmptyOutput,
		"campaign turn returned empty output",
	)
)

func errRunnerUnavailable() error {
	return apperrors.New(apperrors.CodeAIOrchestrationUnavailable, "campaign turn runner is not configured")
}

func errPromptBuilderUnavailable() error {
	return apperrors.New(apperrors.CodeAIOrchestrationUnavailable, "campaign turn prompt builder is not configured")
}

func errInvalidInput(message string) error {
	return apperrors.New(apperrors.CodeAIOrchestrationInvalidInput, message)
}

func errPromptBuild(cause error) error {
	return wrapCampaignTurnFailure(
		apperrors.CodeAIOrchestrationPromptBuildFailed,
		"campaign turn prompt build failed",
		cause,
	)
}

func errExecution(cause error) error {
	return wrapCampaignTurnFailure(
		apperrors.CodeAIOrchestrationExecutionFailed,
		"campaign turn execution failed",
		cause,
	)
}

func wrapCampaignTurnFailure(code apperrors.Code, message string, cause error) error {
	switch {
	case errors.Is(cause, context.DeadlineExceeded):
		return apperrors.Wrap(apperrors.CodeAIOrchestrationTimedOut, "campaign turn timed out", cause)
	case errors.Is(cause, context.Canceled):
		return apperrors.Wrap(apperrors.CodeAIOrchestrationCanceled, "campaign turn canceled", cause)
	default:
		return apperrors.Wrap(code, message, cause)
	}
}
