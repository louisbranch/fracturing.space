package state

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/state/campaign"
	"github.com/louisbranch/fracturing.space/internal/state/event"
	"github.com/louisbranch/fracturing.space/internal/state/projection"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"github.com/louisbranch/fracturing.space/internal/systems/daggerheart"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SnapshotService implements the state.v1.SnapshotService gRPC API.
type SnapshotService struct {
	statev1.UnimplementedSnapshotServiceServer
	stores Stores
}

// NewSnapshotService creates a SnapshotService with default dependencies.
func NewSnapshotService(stores Stores) *SnapshotService {
	return &SnapshotService{
		stores: stores,
	}
}

// GetSnapshot returns the snapshot state for a campaign.
func (s *SnapshotService) GetSnapshot(ctx context.Context, in *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
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

	// Get Daggerheart snapshot state (GM Fear)
	dhSnapshot, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "get daggerheart snapshot: %v", err)
	}

	// Get all character states for this campaign
	charPage, err := s.stores.Character.ListCharacters(ctx, campaignID, 100, "")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list characters: %v", err)
	}

	characterStates := make([]*statev1.CharacterState, 0, len(charPage.Characters))
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

	return &statev1.GetSnapshotResponse{
		Snapshot: &statev1.Snapshot{
			CampaignId:      campaignID,
			CharacterStates: characterStates,
			SystemSnapshot: &statev1.Snapshot_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartSnapshot{
					GmFear: int32(dhSnapshot.GMFear),
				},
			},
		},
	}, nil
}

// PatchCharacterState patches a character's state (system-specific state like HP, Hope, Stress).
func (s *SnapshotService) PatchCharacterState(ctx context.Context, in *statev1.PatchCharacterStateRequest) (*statev1.PatchCharacterStateResponse, error) {
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

		// Apply Hope
		hope := int(dhPatch.Hope)
		if hope < daggerheart.HopeMin || hope > daggerheart.HopeMax {
			return nil, status.Errorf(codes.InvalidArgument, "hope %d exceeds range %d..%d", hope, daggerheart.HopeMin, daggerheart.HopeMax)
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

		hpBefore := dhState.Hp
		hpAfter := hp
		payload := event.CharacterStateChangedPayload{
			CharacterID: characterID,
			HpBefore:    &hpBefore,
			HpAfter:     &hpAfter,
			SystemState: map[string]any{
				"daggerheart": map[string]any{
					"hope_before":   dhState.Hope,
					"hope_after":    hope,
					"stress_before": dhState.Stress,
					"stress_after":  stress,
				},
			},
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
			CampaignID:   campaignID,
			Timestamp:    time.Now().UTC(),
			Type:         event.TypeCharacterStateChanged,
			SessionID:    grpcmeta.SessionIDFromContext(ctx),
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			ActorType:    actorType,
			ActorID:      actorID,
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  payloadJSON,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "append event: %v", err)
		}

		applier := projection.Applier{Campaign: s.stores.Campaign, Daggerheart: s.stores.Daggerheart}
		if err := applier.Apply(ctx, stored); err != nil {
			return nil, status.Errorf(codes.Internal, "apply event: %v", err)
		}

		dhState, err = s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load daggerheart character state: %v", err)
		}
	}

	return &statev1.PatchCharacterStateResponse{
		State: daggerheartStateToProto(campaignID, characterID, dhState),
	}, nil
}

// UpdateSnapshotState updates the system-specific snapshot state.
func (s *SnapshotService) UpdateSnapshotState(ctx context.Context, in *statev1.UpdateSnapshotStateRequest) (*statev1.UpdateSnapshotStateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update snapshot state request is required")
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

		payload := event.GMFearChangedPayload{
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
			CampaignID:   campaignID,
			Timestamp:    time.Now().UTC(),
			Type:         event.TypeGMFearChanged,
			SessionID:    grpcmeta.SessionIDFromContext(ctx),
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			ActorType:    actorType,
			ActorID:      actorID,
			EntityType:   "snapshot",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "append event: %v", err)
		}

		applier := projection.Applier{Campaign: s.stores.Campaign, Daggerheart: s.stores.Daggerheart}
		if err := applier.Apply(ctx, stored); err != nil {
			return nil, status.Errorf(codes.Internal, "apply event: %v", err)
		}

		dhSnapshot, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load daggerheart snapshot: %v", err)
		}

		return &statev1.UpdateSnapshotStateResponse{
			Snapshot: &statev1.Snapshot{
				CampaignId: campaignID,
				SystemSnapshot: &statev1.Snapshot_Daggerheart{
					Daggerheart: &daggerheartv1.DaggerheartSnapshot{
						GmFear: int32(dhSnapshot.GMFear),
					},
				},
			},
		}, nil
	}

	return nil, status.Error(codes.InvalidArgument, "no system snapshot update provided")
}
