package creationworkflow

import (
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/characterworkflow"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func progressFromDaggerheart(progress daggerheart.CreationProgress) characterworkflow.Progress {
	steps := make([]characterworkflow.StepProgress, 0, len(progress.Steps))
	for _, step := range progress.Steps {
		steps = append(steps, characterworkflow.StepProgress{
			Step:     step.Step,
			Key:      step.Key,
			Complete: step.Complete,
		})
	}
	return characterworkflow.Progress{
		Steps:        steps,
		NextStep:     progress.NextStep,
		Ready:        progress.Ready,
		UnmetReasons: append([]string(nil), progress.UnmetReasons...),
	}
}

// HandleWorkflowError maps domain errors to gRPC status errors for workflow
// endpoints. Unknown errors pass through unchanged.
func HandleWorkflowError(err error) error {
	if err == nil {
		return nil
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		return grpcerror.HandleDomainError(err)
	}
	return err
}
