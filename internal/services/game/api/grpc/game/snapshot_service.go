package game

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
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
	dhSnapshot, err := s.stores.SystemStores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
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
		dhState, err := s.stores.SystemStores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, ch.ID)
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	characterID, dhState, err := newSnapshotApplication(s).PatchCharacterState(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	dhSnapshot, err := newSnapshotApplication(s).UpdateSnapshotState(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
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
