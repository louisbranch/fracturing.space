package game

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type snapshotApplication struct {
	stores Stores
}

func newSnapshotApplication(service *SnapshotService) snapshotApplication {
	return snapshotApplication{stores: service.stores}
}

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

	// Get existing Daggerheart state
	dhState, err := a.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return "", storage.DaggerheartCharacterState{}, err
	}

	// Get Daggerheart profile for validation
	dhProfile, err := a.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "get daggerheart profile: %v", err)
	}

	// Apply Daggerheart-specific patches (including HP)
	if dhPatch := in.GetDaggerheart(); dhPatch != nil {
		// Apply HP
		hp := int(dhPatch.Hp)
		hpMax := dhProfile.HpMax
		if hpMax == 0 {
			hpMax = 6 // Default
		}
		if hp < 0 || hp > hpMax {
			return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.InvalidArgument, "hp %d exceeds range 0..%d", hp, hpMax)
		}

		// Apply Hope Max
		hopeMax := int(dhPatch.HopeMax)
		if hopeMax == 0 {
			hopeMax = dhState.HopeMax
			if hopeMax == 0 {
				hopeMax = daggerheart.HopeMax
			}
		}
		if hopeMax < daggerheart.HopeMin || hopeMax > daggerheart.HopeMax {
			return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.InvalidArgument, "hope_max %d exceeds range %d..%d", hopeMax, daggerheart.HopeMin, daggerheart.HopeMax)
		}

		// Apply Hope
		hope := int(dhPatch.Hope)
		if hope < daggerheart.HopeMin || hope > hopeMax {
			return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.InvalidArgument, "hope %d exceeds range %d..%d", hope, daggerheart.HopeMin, hopeMax)
		}

		// Apply Stress
		stress := int(dhPatch.Stress)
		stressMax := dhProfile.StressMax
		if stressMax == 0 {
			stressMax = 6 // Default
		}
		if stress < 0 || stress > stressMax {
			return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.InvalidArgument, "stress %d exceeds range 0..%d", stress, stressMax)
		}

		// Apply Armor
		armor := int(dhPatch.Armor)
		armorMax := dhProfile.ArmorMax
		if armorMax < 0 {
			armorMax = 0
		}
		if armor < 0 || armor > armorMax {
			return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.InvalidArgument, "armor %d exceeds range 0..%d", armor, armorMax)
		}

		var normalizedConditions []string
		conditionPatch := dhPatch.Conditions != nil
		if conditionPatch {
			conditions, err := daggerheartConditionsFromProto(dhPatch.Conditions)
			if err != nil {
				return "", storage.DaggerheartCharacterState{}, status.Error(codes.InvalidArgument, err.Error())
			}
			normalizedConditions, err = daggerheart.NormalizeConditions(conditions)
			if err != nil {
				return "", storage.DaggerheartCharacterState{}, status.Error(codes.InvalidArgument, err.Error())
			}
		}

		lifeState := dhState.LifeState
		if dhPatch.LifeState != daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED {
			var err error
			lifeState, err = daggerheartLifeStateFromProto(dhPatch.LifeState)
			if err != nil {
				return "", storage.DaggerheartCharacterState{}, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		if lifeState == "" {
			lifeState = daggerheart.LifeStateAlive
		}

		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		actorType := event.ActorTypeSystem
		if actorID != "" {
			actorType = event.ActorTypeGM
		}

		hpBefore := dhState.Hp
		hpAfter := hp
		hopeBefore := dhState.Hope
		hopeAfter := hope
		hopeMaxBefore := dhState.HopeMax
		hopeMaxAfter := hopeMax
		stressBefore := dhState.Stress
		stressAfter := stress
		armorBefore := dhState.Armor
		armorAfter := armor
		lifeStateBefore := dhState.LifeState
		if lifeStateBefore == "" {
			lifeStateBefore = daggerheart.LifeStateAlive
		}
		lifeStateAfter := lifeState
		payload := daggerheart.CharacterStatePatchedPayload{
			CharacterID:     characterID,
			HPBefore:        &hpBefore,
			HPAfter:         &hpAfter,
			HopeBefore:      &hopeBefore,
			HopeAfter:       &hopeAfter,
			HopeMaxBefore:   &hopeMaxBefore,
			HopeMaxAfter:    &hopeMaxAfter,
			StressBefore:    &stressBefore,
			StressAfter:     &stressAfter,
			ArmorBefore:     &armorBefore,
			ArmorAfter:      &armorAfter,
			LifeStateBefore: &lifeStateBefore,
			LifeStateAfter:  &lifeStateAfter,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "encode payload: %v", err)
		}

		applier := a.stores.Applier()
		requestID := grpcmeta.RequestIDFromContext(ctx)
		invocationID := grpcmeta.InvocationIDFromContext(ctx)
		if a.stores.Domain == nil {
			return "", storage.DaggerheartCharacterState{}, status.Error(codes.Internal, "domain engine is not configured")
		}
		actorTypeForCommand := command.ActorTypeSystem
		switch actorType {
		case event.ActorTypeParticipant:
			actorTypeForCommand = command.ActorTypeParticipant
		case event.ActorTypeGM:
			actorTypeForCommand = command.ActorTypeGM
		}
		result, err := a.stores.Domain.Execute(ctx, command.Command{
			CampaignID:    campaignID,
			Type:          command.Type("sys.daggerheart.action.character_state.patch"),
			ActorType:     actorTypeForCommand,
			ActorID:       actorID,
			SessionID:     grpcmeta.SessionIDFromContext(ctx),
			RequestID:     requestID,
			InvocationID:  invocationID,
			EntityType:    "character",
			EntityID:      characterID,
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   payloadJSON,
		})
		if err != nil {
			return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
		}
		if len(result.Decision.Rejections) > 0 {
			return "", storage.DaggerheartCharacterState{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
		}
		if len(result.Decision.Events) == 0 {
			return "", storage.DaggerheartCharacterState{}, status.Error(codes.Internal, "character state patch did not emit an event")
		}
		for _, evt := range result.Decision.Events {
			if err := applier.Apply(ctx, evt); err != nil {
				return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "apply event: %v", err)
			}
		}
		if !conditionPatch {
			if err := applyStressVulnerableCondition(ctx, a.stores, campaignID, grpcmeta.SessionIDFromContext(ctx), characterID, dhState.Conditions, stressBefore, stressAfter, stressMax, actorType, actorID); err != nil {
				return "", storage.DaggerheartCharacterState{}, err
			}
		}

		dhState, err = a.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "load daggerheart character state: %v", err)
		}

		if conditionPatch {
			normalizedBefore, err := daggerheart.NormalizeConditions(dhState.Conditions)
			if err != nil {
				return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "invalid stored conditions: %v", err)
			}
			if !daggerheart.ConditionsEqual(normalizedBefore, normalizedConditions) {
				added, removed := daggerheart.DiffConditions(normalizedBefore, normalizedConditions)
				conditionPayload := daggerheart.ConditionChangedPayload{
					CharacterID:      characterID,
					ConditionsBefore: normalizedBefore,
					ConditionsAfter:  normalizedConditions,
					Added:            added,
					Removed:          removed,
				}
				conditionJSON, err := json.Marshal(conditionPayload)
				if err != nil {
					return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "encode condition payload: %v", err)
				}

				if a.stores.Domain == nil {
					return "", storage.DaggerheartCharacterState{}, status.Error(codes.Internal, "domain engine is not configured")
				}
				actorTypeForCommand := command.ActorTypeSystem
				switch actorType {
				case event.ActorTypeParticipant:
					actorTypeForCommand = command.ActorTypeParticipant
				case event.ActorTypeGM:
					actorTypeForCommand = command.ActorTypeGM
				}
				result, err := a.stores.Domain.Execute(ctx, command.Command{
					CampaignID:    campaignID,
					Type:          command.Type("sys.daggerheart.action.condition.change"),
					ActorType:     actorTypeForCommand,
					ActorID:       actorID,
					SessionID:     grpcmeta.SessionIDFromContext(ctx),
					RequestID:     grpcmeta.RequestIDFromContext(ctx),
					InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
					EntityType:    "character",
					EntityID:      characterID,
					SystemID:      daggerheart.SystemID,
					SystemVersion: daggerheart.SystemVersion,
					PayloadJSON:   conditionJSON,
				})
				if err != nil {
					return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
				}
				if len(result.Decision.Rejections) > 0 {
					return "", storage.DaggerheartCharacterState{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
				}
				if len(result.Decision.Events) == 0 {
					return "", storage.DaggerheartCharacterState{}, status.Error(codes.Internal, "condition change did not emit an event")
				}
				for _, evt := range result.Decision.Events {
					if err := applier.Apply(ctx, evt); err != nil {
						return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "apply event: %v", err)
					}
				}

				dhState, err = a.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
				if err != nil {
					return "", storage.DaggerheartCharacterState{}, status.Errorf(codes.Internal, "load daggerheart character state: %v", err)
				}
			}
		}
	}

	return characterID, dhState, nil
}

func (a snapshotApplication) UpdateSnapshotState(ctx context.Context, campaignID string, in *campaignv1.UpdateSnapshotStateRequest) (storage.DaggerheartSnapshot, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.DaggerheartSnapshot{}, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.DaggerheartSnapshot{}, err
	}

	// Handle Daggerheart snapshot update
	if dhUpdate := in.GetDaggerheart(); dhUpdate != nil {
		gmFear := int(dhUpdate.GetGmFear())
		if gmFear < daggerheart.GMFearMin || gmFear > daggerheart.GMFearMax {
			return storage.DaggerheartSnapshot{}, status.Errorf(codes.InvalidArgument, "gm_fear %d exceeds range %d..%d", gmFear, daggerheart.GMFearMin, daggerheart.GMFearMax)
		}

		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		applier := a.stores.Applier()
		requestID := grpcmeta.RequestIDFromContext(ctx)
		invocationID := grpcmeta.InvocationIDFromContext(ctx)
		if a.stores.Domain == nil {
			return storage.DaggerheartSnapshot{}, status.Error(codes.Internal, "domain engine is not configured")
		}
		after := gmFear
		payload := daggerheart.GMFearSetPayload{After: &after}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return storage.DaggerheartSnapshot{}, status.Errorf(codes.Internal, "encode payload: %v", err)
		}
		actorTypeForCommand := command.ActorTypeSystem
		if actorID != "" {
			actorTypeForCommand = command.ActorTypeGM
		}
		result, err := a.stores.Domain.Execute(ctx, command.Command{
			CampaignID:    campaignID,
			Type:          command.Type("sys.daggerheart.action.gm_fear.set"),
			ActorType:     actorTypeForCommand,
			ActorID:       actorID,
			SessionID:     grpcmeta.SessionIDFromContext(ctx),
			RequestID:     requestID,
			InvocationID:  invocationID,
			EntityType:    "campaign",
			EntityID:      campaignID,
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   payloadJSON,
		})
		if err != nil {
			return storage.DaggerheartSnapshot{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
		}
		if len(result.Decision.Rejections) > 0 {
			return storage.DaggerheartSnapshot{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
		}
		if len(result.Decision.Events) == 0 {
			return storage.DaggerheartSnapshot{}, status.Error(codes.Internal, "gm fear update did not emit an event")
		}
		for _, evt := range result.Decision.Events {
			if err := applier.Apply(ctx, evt); err != nil {
				return storage.DaggerheartSnapshot{}, status.Errorf(codes.Internal, "apply event: %v", err)
			}
		}

		dhSnapshot, err := a.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return storage.DaggerheartSnapshot{}, status.Errorf(codes.Internal, "load daggerheart snapshot: %v", err)
		}

		return dhSnapshot, nil
	}

	return storage.DaggerheartSnapshot{}, status.Error(codes.InvalidArgument, "no system snapshot update provided")
}

func applyStressVulnerableCondition(
	ctx context.Context,
	stores Stores,
	campaignID string,
	sessionID string,
	characterID string,
	conditions []string,
	stressBefore int,
	stressAfter int,
	stressMax int,
	actorType event.ActorType,
	actorID string,
) error {
	if stores.Domain == nil || stores.Daggerheart == nil {
		return status.Error(codes.Internal, "domain engine or daggerheart store is not configured")
	}
	if stressMax <= 0 {
		return nil
	}
	if stressBefore == stressAfter {
		return nil
	}
	shouldAdd := stressBefore < stressMax && stressAfter == stressMax
	shouldRemove := stressBefore == stressMax && stressAfter < stressMax
	if !shouldAdd && !shouldRemove {
		return nil
	}

	normalized, err := daggerheart.NormalizeConditions(conditions)
	if err != nil {
		return status.Errorf(codes.Internal, "invalid stored conditions: %v", err)
	}
	hasVulnerable := false
	for _, value := range normalized {
		if value == daggerheart.ConditionVulnerable {
			hasVulnerable = true
			break
		}
	}
	if shouldAdd && hasVulnerable {
		return nil
	}
	if shouldRemove && !hasVulnerable {
		return nil
	}

	afterSet := make(map[string]struct{}, len(normalized)+1)
	for _, value := range normalized {
		afterSet[value] = struct{}{}
	}
	if shouldAdd {
		afterSet[daggerheart.ConditionVulnerable] = struct{}{}
	}
	if shouldRemove {
		delete(afterSet, daggerheart.ConditionVulnerable)
	}
	afterList := make([]string, 0, len(afterSet))
	for value := range afterSet {
		afterList = append(afterList, value)
	}
	after, err := daggerheart.NormalizeConditions(afterList)
	if err != nil {
		return status.Errorf(codes.Internal, "invalid condition set: %v", err)
	}
	added, removed := daggerheart.DiffConditions(normalized, after)
	if len(added) == 0 && len(removed) == 0 {
		return nil
	}

	payload := daggerheart.ConditionChangedPayload{
		CharacterID:      characterID,
		ConditionsBefore: normalized,
		ConditionsAfter:  after,
		Added:            added,
		Removed:          removed,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode condition payload: %v", err)
	}

	actorTypeForCommand := command.ActorTypeSystem
	switch actorType {
	case event.ActorTypeParticipant:
		actorTypeForCommand = command.ActorTypeParticipant
	case event.ActorTypeGM:
		actorTypeForCommand = command.ActorTypeGM
	}
	result, err := stores.Domain.Execute(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          command.Type("sys.daggerheart.action.condition.change"),
		ActorType:     actorTypeForCommand,
		ActorID:       actorID,
		SessionID:     sessionID,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "execute domain command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		return status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
	}
	if len(result.Decision.Events) == 0 {
		return status.Error(codes.Internal, "condition change did not emit an event")
	}
	applier := stores.Applier()
	for _, evt := range result.Decision.Events {
		if err := applier.Apply(ctx, evt); err != nil {
			return status.Errorf(codes.Internal, "apply condition event: %v", err)
		}
	}

	return nil
}
