package game

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SnapshotService implements the game.v1.SnapshotService gRPC API.
type SnapshotService struct {
	campaignv1.UnimplementedSnapshotServiceServer
	stores Stores
}

// NewSnapshotService creates a SnapshotService with default dependencies.
func NewSnapshotService(stores Stores) *SnapshotService {
	return &SnapshotService{
		stores: stores,
	}
}

// GetSnapshot returns the snapshot projection for a campaign.
func (s *SnapshotService) GetSnapshot(ctx context.Context, in *campaignv1.GetSnapshotRequest) (*campaignv1.GetSnapshotResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get snapshot request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}

	// Get Daggerheart snapshot projection (GM Fear)
	dhSnapshot, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "get daggerheart snapshot: %v", err)
	}

	// Get all character states for this campaign
	charPage, err := s.stores.Character.ListCharacters(ctx, campaignID, 100, "")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list characters: %v", err)
	}

	characterStates := make([]*campaignv1.CharacterState, 0, len(charPage.Characters))
	for _, ch := range charPage.Characters {
		// Get Daggerheart-specific state (includes HP)
		dhState, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, ch.ID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, status.Errorf(codes.Internal, "get daggerheart character state: %v", err)
		}

		characterStates = append(characterStates, daggerheartStateToProto(campaignID, ch.ID, dhState))
	}

	return &campaignv1.GetSnapshotResponse{
		Snapshot: &campaignv1.Snapshot{
			CampaignId:      campaignID,
			CharacterStates: characterStates,
			SystemSnapshot: &campaignv1.Snapshot_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartSnapshot{
					GmFear:                int32(dhSnapshot.GMFear),
					ConsecutiveShortRests: int32(dhSnapshot.ConsecutiveShortRests),
				},
			},
		},
	}, nil
}

// PatchCharacterState patches a character's state (system-specific state like HP, Hope, Stress).
func (s *SnapshotService) PatchCharacterState(ctx context.Context, in *campaignv1.PatchCharacterStateRequest) (*campaignv1.PatchCharacterStateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "patch character state request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	// Get existing Daggerheart state
	dhState, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	// Get Daggerheart profile for validation
	dhProfile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "get daggerheart profile: %v", err)
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
			return nil, status.Errorf(codes.InvalidArgument, "hp %d exceeds range 0..%d", hp, hpMax)
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
			return nil, status.Errorf(codes.InvalidArgument, "hope_max %d exceeds range %d..%d", hopeMax, daggerheart.HopeMin, daggerheart.HopeMax)
		}

		// Apply Hope
		hope := int(dhPatch.Hope)
		if hope < daggerheart.HopeMin || hope > hopeMax {
			return nil, status.Errorf(codes.InvalidArgument, "hope %d exceeds range %d..%d", hope, daggerheart.HopeMin, hopeMax)
		}

		// Apply Stress
		stress := int(dhPatch.Stress)
		stressMax := dhProfile.StressMax
		if stressMax == 0 {
			stressMax = 6 // Default
		}
		if stress < 0 || stress > stressMax {
			return nil, status.Errorf(codes.InvalidArgument, "stress %d exceeds range 0..%d", stress, stressMax)
		}

		// Apply Armor
		armor := int(dhPatch.Armor)
		armorMax := dhProfile.ArmorMax
		if armorMax < 0 {
			armorMax = 0
		}
		if armor < 0 || armor > armorMax {
			return nil, status.Errorf(codes.InvalidArgument, "armor %d exceeds range 0..%d", armor, armorMax)
		}

		var normalizedConditions []string
		conditionPatch := dhPatch.Conditions != nil
		if conditionPatch {
			conditions, err := daggerheartConditionsFromProto(dhPatch.Conditions)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			normalizedConditions, err = daggerheart.NormalizeConditions(conditions)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}

		lifeState := dhState.LifeState
		if dhPatch.LifeState != daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED {
			var err error
			lifeState, err = daggerheartLifeStateFromProto(dhPatch.LifeState)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
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
			HpBefore:        &hpBefore,
			HpAfter:         &hpAfter,
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
			return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
		}

		stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
			CampaignID:    campaignID,
			Timestamp:     time.Now().UTC(),
			Type:          daggerheart.EventTypeCharacterStatePatched,
			SessionID:     grpcmeta.SessionIDFromContext(ctx),
			RequestID:     grpcmeta.RequestIDFromContext(ctx),
			InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
			ActorType:     actorType,
			ActorID:       actorID,
			EntityType:    "character",
			EntityID:      characterID,
			SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   payloadJSON,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "append event: %v", err)
		}

		adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
		if err := adapter.ApplyEvent(ctx, stored); err != nil {
			return nil, status.Errorf(codes.Internal, "apply event: %v", err)
		}

		dhState, err = s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load daggerheart character state: %v", err)
		}

		if conditionPatch {
			normalizedBefore, err := daggerheart.NormalizeConditions(dhState.Conditions)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "invalid stored conditions: %v", err)
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
					return nil, status.Errorf(codes.Internal, "encode condition payload: %v", err)
				}

				stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
					CampaignID:    campaignID,
					Timestamp:     time.Now().UTC(),
					Type:          daggerheart.EventTypeConditionChanged,
					SessionID:     grpcmeta.SessionIDFromContext(ctx),
					RequestID:     grpcmeta.RequestIDFromContext(ctx),
					InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
					ActorType:     actorType,
					ActorID:       actorID,
					EntityType:    "character",
					EntityID:      characterID,
					SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
					SystemVersion: daggerheart.SystemVersion,
					PayloadJSON:   conditionJSON,
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "append event: %v", err)
				}

				if err := adapter.ApplyEvent(ctx, stored); err != nil {
					return nil, status.Errorf(codes.Internal, "apply event: %v", err)
				}

				dhState, err = s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "load daggerheart character state: %v", err)
				}
			}
		}
	}

	return &campaignv1.PatchCharacterStateResponse{
		State: daggerheartStateToProto(campaignID, characterID, dhState),
	}, nil
}

// UpdateSnapshotState updates the system-specific snapshot projection.
func (s *SnapshotService) UpdateSnapshotState(ctx context.Context, in *campaignv1.UpdateSnapshotStateRequest) (*campaignv1.UpdateSnapshotStateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update snapshot projection request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}

	// Handle Daggerheart snapshot update
	if dhUpdate := in.GetDaggerheart(); dhUpdate != nil {
		gmFear := int(dhUpdate.GetGmFear())
		if gmFear < daggerheart.GMFearMin || gmFear > daggerheart.GMFearMax {
			return nil, status.Errorf(codes.InvalidArgument, "gm_fear %d exceeds range %d..%d", gmFear, daggerheart.GMFearMin, daggerheart.GMFearMax)
		}

		before := 0
		current, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, status.Errorf(codes.Internal, "get daggerheart snapshot: %v", err)
		}
		if err == nil {
			before = current.GMFear
		}

		payload := daggerheart.GMFearChangedPayload{
			Before: before,
			After:  gmFear,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
		}

		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		actorType := event.ActorTypeSystem
		if actorID != "" {
			actorType = event.ActorTypeGM
		}

		stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
			CampaignID:    campaignID,
			Timestamp:     time.Now().UTC(),
			Type:          daggerheart.EventTypeGMFearChanged,
			SessionID:     grpcmeta.SessionIDFromContext(ctx),
			RequestID:     grpcmeta.RequestIDFromContext(ctx),
			InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
			ActorType:     actorType,
			ActorID:       actorID,
			EntityType:    "campaign",
			EntityID:      campaignID,
			SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   payloadJSON,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "append event: %v", err)
		}

		adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
		if err := adapter.ApplyEvent(ctx, stored); err != nil {
			return nil, status.Errorf(codes.Internal, "apply event: %v", err)
		}

		dhSnapshot, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load daggerheart snapshot: %v", err)
		}

		return &campaignv1.UpdateSnapshotStateResponse{
			Snapshot: &campaignv1.Snapshot{
				CampaignId: campaignID,
				SystemSnapshot: &campaignv1.Snapshot_Daggerheart{
					Daggerheart: &daggerheartv1.DaggerheartSnapshot{
						GmFear:                int32(dhSnapshot.GMFear),
						ConsecutiveShortRests: int32(dhSnapshot.ConsecutiveShortRests),
					},
				},
			},
		}, nil
	}

	return nil, status.Error(codes.InvalidArgument, "no system snapshot update provided")
}
