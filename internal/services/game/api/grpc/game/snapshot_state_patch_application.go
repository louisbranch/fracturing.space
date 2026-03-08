package game

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a snapshotApplication) PatchCharacterState(ctx context.Context, campaignID string, in *campaignv1.PatchCharacterStateRequest) (string, storage.DaggerheartCharacterState, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", storage.DaggerheartCharacterState{}, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return "", storage.DaggerheartCharacterState{}, err
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return "", storage.DaggerheartCharacterState{}, status.Error(codes.InvalidArgument, "character id is required")
	}
	if _, err := requireCharacterMutationPolicy(ctx, a.stores, c, characterID); err != nil {
		return "", storage.DaggerheartCharacterState{}, err
	}

	// Get existing Daggerheart state
	dhState, err := a.stores.SystemStores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return "", storage.DaggerheartCharacterState{}, err
	}

	// Get Daggerheart profile for validation
	dhProfile, err := a.stores.SystemStores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "get daggerheart profile: %v", err)
	}

	// Apply Daggerheart-specific patches (including HP)
	if dhPatch := in.GetDaggerheart(); dhPatch != nil {
		patch, err := buildDaggerheartCharacterStatePatch(dhState, dhProfile, dhPatch)
		if err != nil {
			return "", storage.DaggerheartCharacterState{}, err
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
			a.stores,
			campaignID,
			characterID,
			actorType,
			actorID,
			patch.payload(characterID, dhState),
		); err != nil {
			return "", storage.DaggerheartCharacterState{}, err
		}
		if !patch.conditionPatch {
			if err := applyStressVulnerableCondition(
				ctx,
				a.stores,
				campaignID,
				grpcmeta.SessionIDFromContext(ctx),
				characterID,
				dhState.Conditions,
				dhState.Stress,
				patch.stress,
				patch.stressMax,
				actorType,
				actorID,
			); err != nil {
				return "", storage.DaggerheartCharacterState{}, err
			}
		}

		dhState, err = a.loadDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			return "", storage.DaggerheartCharacterState{}, err
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
				return "", storage.DaggerheartCharacterState{}, err
			}
		}
	}

	return characterID, dhState, nil
}

func (a snapshotApplication) applyConditionPatchIfChanged(
	ctx context.Context,
	campaignID string,
	characterID string,
	current storage.DaggerheartCharacterState,
	normalizedAfter []string,
	actorType event.ActorType,
	actorID string,
) (storage.DaggerheartCharacterState, error) {
	normalizedBefore, err := daggerheart.NormalizeConditions(current.Conditions)
	if err != nil {
		return storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "invalid stored conditions: %v", err)
	}
	if daggerheart.ConditionsEqual(normalizedBefore, normalizedAfter) {
		return current, nil
	}

	added, removed := daggerheart.DiffConditions(normalizedBefore, normalizedAfter)
	conditionPayload := daggerheart.ConditionChangePayload{
		CharacterID:      ids.CharacterID(characterID),
		ConditionsBefore: normalizedBefore,
		ConditionsAfter:  normalizedAfter,
		Added:            added,
		Removed:          removed,
	}
	if err := executeDaggerheartConditionChangeCommand(
		ctx,
		a.stores,
		campaignID,
		characterID,
		actorType,
		actorID,
		grpcmeta.SessionIDFromContext(ctx),
		conditionPayload,
		"apply event",
	); err != nil {
		return storage.DaggerheartCharacterState{}, err
	}

	return a.loadDaggerheartCharacterState(ctx, campaignID, characterID)
}

func (a snapshotApplication) loadDaggerheartCharacterState(ctx context.Context, campaignID string, characterID string) (storage.DaggerheartCharacterState, error) {
	dhState, err := a.stores.SystemStores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "load daggerheart character state: %v", err)
	}
	return dhState, nil
}
