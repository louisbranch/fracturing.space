package creationworkflow

import (
	"context"
	"errors"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/characterworkflow"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreationWorkflowProvider implements the character creation workflow for
// the Daggerheart game system. It satisfies characterworkflow.Provider.
type CreationWorkflowProvider struct{}

func (CreationWorkflowProvider) GetProgress(ctx context.Context, deps characterworkflow.CreationDeps, campaignContext characterworkflow.CampaignContext, characterID string) (characterworkflow.Progress, error) {
	if err := campaign.ValidateCampaignOperation(campaignContext.Status, campaign.CampaignOpRead); err != nil {
		return characterworkflow.Progress{}, err
	}
	if err := deps.RequireReadPolicy(ctx, campaignContext); err != nil {
		return characterworkflow.Progress{}, err
	}

	if _, err := deps.GetCharacterRecord(ctx, campaignContext.ID, characterID); err != nil {
		return characterworkflow.Progress{}, err
	}

	profile, err := deps.GetCharacterSystemProfile(ctx, campaignContext.ID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return characterworkflow.Progress{}, grpcerror.Internal("get daggerheart profile", err)
		}
		profile = projectionstore.DaggerheartCharacterProfile{CampaignID: campaignContext.ID, CharacterID: characterID}
	}

	progress := daggerheart.EvaluateCreationProgress(daggerheart.CharacterProfileFromStorage(profile).CreationProfile())
	return progressFromDaggerheart(progress), nil
}

func (CreationWorkflowProvider) ApplyStep(ctx context.Context, deps characterworkflow.CreationDeps, campaignContext characterworkflow.CampaignContext, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, characterworkflow.Progress, error) {
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, characterworkflow.Progress{}, err
	}
	stepInput := in.GetDaggerheart()
	if stepInput == nil {
		return nil, characterworkflow.Progress{}, status.Error(codes.InvalidArgument, "daggerheart step payload is required")
	}
	stepNumber, err := creationStepNumber(stepInput)
	if err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	if err := campaign.ValidateCampaignOperation(campaignContext.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	characterRecord, err := deps.GetCharacterRecord(ctx, campaignContext.ID, characterID)
	if err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	profile, err := deps.GetCharacterSystemProfile(ctx, campaignContext.ID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, characterworkflow.Progress{}, grpcerror.Internal("get daggerheart profile", err)
		}
		profile = defaultProfileForCharacter(campaignContext.ID, characterRecord.Kind)
	} else {
		profile = ensureProfileDefaults(profile, characterRecord.Kind)
	}

	currentProgress := daggerheart.EvaluateCreationProgress(daggerheart.CharacterProfileFromStorage(profile).CreationProfile())
	if currentProgress.Ready {
		return nil, characterworkflow.Progress{}, status.Error(codes.FailedPrecondition, "character creation workflow is already complete")
	}
	if stepNumber != currentProgress.NextStep {
		return nil, characterworkflow.Progress{}, status.Errorf(
			codes.FailedPrecondition,
			"creation step %d is not allowed; expected step %d",
			stepNumber,
			currentProgress.NextStep,
		)
	}

	if err := applyCreationStepInput(ctx, deps.SystemContent(), &profile, stepInput); err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	if err := validateProfile(profile); err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	if err := deps.ExecuteProfileReplace(ctx, campaignContext, characterID, daggerheart.CharacterProfileFromStorage(profile)); err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	nextProgress := daggerheart.EvaluateCreationProgress(daggerheart.CharacterProfileFromStorage(profile).CreationProfile())
	return deps.ProfileToProto(campaignContext.ID, characterID, profile), progressFromDaggerheart(nextProgress), nil
}

func (CreationWorkflowProvider) ApplyWorkflow(ctx context.Context, deps characterworkflow.CreationDeps, campaignContext characterworkflow.CampaignContext, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, characterworkflow.Progress, error) {
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, characterworkflow.Progress{}, err
	}
	workflowInput := in.GetDaggerheart()
	if workflowInput == nil {
		return nil, characterworkflow.Progress{}, status.Error(codes.InvalidArgument, "daggerheart workflow payload is required")
	}

	if err := campaign.ValidateCampaignOperation(campaignContext.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	characterRecord, err := deps.GetCharacterRecord(ctx, campaignContext.ID, characterID)
	if err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	profile, err := deps.GetCharacterSystemProfile(ctx, campaignContext.ID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, characterworkflow.Progress{}, grpcerror.Internal("get daggerheart profile", err)
		}
		profile = defaultProfileForCharacter(campaignContext.ID, characterRecord.Kind)
	} else {
		profile = ensureProfileDefaults(profile, characterRecord.Kind)
	}

	profile = resetCreationWorkflowFields(profile)

	steps, err := creationStepSequenceFromWorkflowInput(workflowInput)
	if err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	for idx, stepInput := range steps {
		expectedStep := int32(idx + 1)
		currentProgress := daggerheart.EvaluateCreationProgress(daggerheart.CharacterProfileFromStorage(profile).CreationProfile())
		if currentProgress.Ready {
			return nil, characterworkflow.Progress{}, status.Error(codes.FailedPrecondition, "character creation workflow is already complete")
		}
		if currentProgress.NextStep != expectedStep {
			return nil, characterworkflow.Progress{}, status.Errorf(codes.FailedPrecondition, "creation step %d is not allowed; expected step %d", expectedStep, currentProgress.NextStep)
		}
		stepNumber, err := creationStepNumber(stepInput)
		if err != nil {
			return nil, characterworkflow.Progress{}, err
		}
		if stepNumber != expectedStep {
			return nil, characterworkflow.Progress{}, status.Errorf(codes.InvalidArgument, "daggerheart workflow payload contains out-of-order step %d at position %d", stepNumber, idx+1)
		}
		if err := applyCreationStepInput(ctx, deps.SystemContent(), &profile, stepInput); err != nil {
			return nil, characterworkflow.Progress{}, err
		}
		if err := validateProfile(profile); err != nil {
			return nil, characterworkflow.Progress{}, err
		}
	}

	finalProgress := daggerheart.EvaluateCreationProgress(daggerheart.CharacterProfileFromStorage(profile).CreationProfile())
	if !finalProgress.Ready {
		if len(finalProgress.UnmetReasons) > 0 {
			return nil, characterworkflow.Progress{}, status.Errorf(codes.FailedPrecondition, "character creation workflow is incomplete: %s", finalProgress.UnmetReasons[0])
		}
		return nil, characterworkflow.Progress{}, status.Error(codes.FailedPrecondition, "character creation workflow is incomplete")
	}

	if err := deps.ExecuteProfileReplace(ctx, campaignContext, characterID, daggerheart.CharacterProfileFromStorage(profile)); err != nil {
		return nil, characterworkflow.Progress{}, err
	}

	return deps.ProfileToProto(campaignContext.ID, characterID, profile), progressFromDaggerheart(finalProgress), nil
}

func (CreationWorkflowProvider) Reset(ctx context.Context, deps characterworkflow.CreationDeps, campaignContext characterworkflow.CampaignContext, characterID string) (characterworkflow.Progress, error) {
	if err := campaign.ValidateCampaignOperation(campaignContext.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return characterworkflow.Progress{}, err
	}

	if _, err := deps.GetCharacterRecord(ctx, campaignContext.ID, characterID); err != nil {
		return characterworkflow.Progress{}, err
	}

	if err := deps.ExecuteProfileDelete(ctx, campaignContext, characterID); err != nil {
		return characterworkflow.Progress{}, err
	}

	return progressFromDaggerheart(daggerheart.EvaluateCreationProgress(daggerheart.CreationProfile{})), nil
}
