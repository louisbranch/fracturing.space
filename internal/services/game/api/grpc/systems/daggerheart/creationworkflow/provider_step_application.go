package creationworkflow

import (
	"context"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// applyCreationStepInput dispatches one creation-step payload to the step-local
// validation and profile mutation rules.
func applyCreationStepInput(ctx context.Context, content contentstore.DaggerheartContentReadStore, profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepInput) error {
	if content == nil {
		return status.Error(codes.Internal, "daggerheart content store is not configured")
	}

	switch step := input.GetStep().(type) {
	case *daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput:
		return applyClassSubclassInput(ctx, content, profile, step.ClassSubclassInput)
	case *daggerheartv1.DaggerheartCreationStepInput_HeritageInput:
		return applyHeritageInput(ctx, content, profile, step.HeritageInput)
	case *daggerheartv1.DaggerheartCreationStepInput_TraitsInput:
		return applyTraitsInput(profile, step.TraitsInput)
	case *daggerheartv1.DaggerheartCreationStepInput_DetailsInput:
		return applyDetailsInput(ctx, content, profile, step.DetailsInput)
	case *daggerheartv1.DaggerheartCreationStepInput_EquipmentInput:
		return applyEquipmentInput(ctx, content, profile, step.EquipmentInput)
	case *daggerheartv1.DaggerheartCreationStepInput_BackgroundInput:
		return applyBackgroundInput(profile, step.BackgroundInput)
	case *daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput:
		return applyExperiencesInput(profile, step.ExperiencesInput)
	case *daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput:
		return applyDomainCardsInput(ctx, content, profile, step.DomainCardsInput)
	case *daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput:
		return applyConnectionsInput(profile, step.ConnectionsInput)
	default:
		return status.Error(codes.InvalidArgument, "daggerheart creation step is required")
	}
}
