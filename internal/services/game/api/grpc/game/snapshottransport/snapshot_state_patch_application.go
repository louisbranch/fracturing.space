package snapshottransport

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func (a snapshotApplication) PatchCharacterState(ctx context.Context, campaignID string, in *campaignv1.PatchCharacterStateRequest) (string, projectionstore.DaggerheartCharacterState, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", projectionstore.DaggerheartCharacterState{}, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return "", projectionstore.DaggerheartCharacterState{}, err
	}

	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return "", projectionstore.DaggerheartCharacterState{}, err
	}
	if _, err := authz.RequireCharacterMutationPolicy(ctx, a.auth, c, characterID); err != nil {
		return "", projectionstore.DaggerheartCharacterState{}, err
	}

	// Get existing Daggerheart state
	dhState, err := a.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return "", projectionstore.DaggerheartCharacterState{}, err
	}

	// Get Daggerheart profile for validation
	dhProfile, err := a.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "get daggerheart profile"); lookupErr != nil {
			return "", projectionstore.DaggerheartCharacterState{}, lookupErr
		}
	}

	// Apply Daggerheart-specific patches (including HP)
	if dhPatch := in.GetDaggerheart(); dhPatch != nil {
		patch, err := buildDaggerheartCharacterStatePatch(dhState, dhProfile, dhPatch)
		if err != nil {
			return "", projectionstore.DaggerheartCharacterState{}, err
		}

		actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
		actorType := event.ActorTypeSystem
		if actorID != "" {
			actorType = event.ActorTypeGM
		}
		if !patch.conditionPatch && patch.stateUnchanged(dhState) {
			// FIXME(telemetry): track no-op character_state patch requests that skip domain commands.
			return characterID, dhState, nil
		}

		if err := applyDaggerheartCharacterStatePatchCommand(
			ctx,
			a.write,
			a.applier,
			campaignID,
			characterID,
			actorType,
			actorID,
			patch.payload(characterID, dhState),
		); err != nil {
			return "", projectionstore.DaggerheartCharacterState{}, err
		}
		if !patch.conditionPatch {
			if err := applyStressVulnerableCondition(
				ctx,
				a.stores.Daggerheart,
				a.write,
				a.applier,
				campaignID,
				grpcmeta.SessionIDFromContext(ctx),
				characterID,
				projectionConditionCodes(dhState.Conditions),
				dhState.Stress,
				patch.stress,
				patch.stressMax,
				actorType,
				actorID,
			); err != nil {
				return "", projectionstore.DaggerheartCharacterState{}, err
			}
		}

		dhState, err = a.loadDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			return "", projectionstore.DaggerheartCharacterState{}, err
		}

		if patch.conditionPatch {
			dhState, err = a.applyConditionPatchIfChanged(
				ctx,
				campaignID,
				characterID,
				dhState,
				patch.normalizedConditions,
				actorType,
				actorID,
			)
			if err != nil {
				return "", projectionstore.DaggerheartCharacterState{}, err
			}
		}
	}

	return characterID, dhState, nil
}

func (a snapshotApplication) applyConditionPatchIfChanged(
	ctx context.Context,
	campaignID string,
	characterID string,
	current projectionstore.DaggerheartCharacterState,
	normalizedAfter []string,
	actorType event.ActorType,
	actorID string,
) (projectionstore.DaggerheartCharacterState, error) {
	normalizedBefore, err := rules.NormalizeConditions(projectionConditionCodes(current.Conditions))
	if err != nil {
		return projectionstore.DaggerheartCharacterState{}, grpcerror.Internal("invalid stored conditions", err)
	}
	if rules.ConditionsEqual(normalizedBefore, normalizedAfter) {
		return current, nil
	}

	added, removed := rules.DiffConditions(normalizedBefore, normalizedAfter)
	beforeStates, err := conditionStatesFromCodes(normalizedBefore)
	if err != nil {
		return projectionstore.DaggerheartCharacterState{}, grpcerror.Internal("invalid stored conditions", err)
	}
	afterStates, err := conditionStatesFromCodes(normalizedAfter)
	if err != nil {
		return projectionstore.DaggerheartCharacterState{}, grpcerror.Internal("invalid condition patch", err)
	}
	addedStates, err := conditionStatesFromCodes(added)
	if err != nil {
		return projectionstore.DaggerheartCharacterState{}, grpcerror.Internal("invalid condition patch", err)
	}
	removedStates, err := conditionStatesFromCodes(removed)
	if err != nil {
		return projectionstore.DaggerheartCharacterState{}, grpcerror.Internal("invalid condition patch", err)
	}
	conditionPayload := daggerheartpayload.ConditionChangePayload{
		CharacterID:      ids.CharacterID(characterID),
		ConditionsBefore: beforeStates,
		ConditionsAfter:  afterStates,
		Added:            addedStates,
		Removed:          removedStates,
	}
	if err := executeDaggerheartConditionChangeCommand(
		ctx,
		a.write,
		a.applier,
		campaignID,
		characterID,
		actorType,
		actorID,
		grpcmeta.SessionIDFromContext(ctx),
		conditionPayload,
		"apply event",
	); err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
	}

	return a.loadDaggerheartCharacterState(ctx, campaignID, characterID)
}

func (a snapshotApplication) loadDaggerheartCharacterState(ctx context.Context, campaignID string, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	dhState, err := a.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return projectionstore.DaggerheartCharacterState{}, grpcerror.Internal("load daggerheart character state", err)
	}
	return dhState, nil
}

func projectionConditionCodes(values []projectionstore.DaggerheartConditionState) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if code == "" {
			code = strings.TrimSpace(value.Standard)
		}
		if code == "" {
			continue
		}
		result = append(result, code)
	}
	return result
}
