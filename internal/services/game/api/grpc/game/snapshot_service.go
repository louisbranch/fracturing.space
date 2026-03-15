package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/charactertransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SnapshotService implements the game.v1.SnapshotService gRPC API.
type SnapshotService struct {
	campaignv1.UnimplementedSnapshotServiceServer
	app snapshotApplication
}

// NewSnapshotService creates a SnapshotService with default dependencies.
func NewSnapshotService(stores Stores) *SnapshotService {
	return &SnapshotService{
		app: newSnapshotApplicationWithDependencies(stores),
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

	readState, err := s.app.GetSnapshot(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	characterStates := make([]*campaignv1.CharacterState, 0, len(readState.characterStates))
	for _, state := range readState.characterStates {
		characterStates = append(characterStates, charactertransport.DaggerheartStateToProto(campaignID, state.CharacterID, state))
	}

	return &campaignv1.GetSnapshotResponse{
		Snapshot: &campaignv1.Snapshot{
			CampaignId:      campaignID,
			CharacterStates: characterStates,
			SystemSnapshot: &campaignv1.Snapshot_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartSnapshot{
					GmFear:                int32(readState.systemState.GMFear),
					ConsecutiveShortRests: int32(readState.systemState.ConsecutiveShortRests),
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

	characterID, dhState, err := s.app.PatchCharacterState(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.PatchCharacterStateResponse{
		State: charactertransport.DaggerheartStateToProto(campaignID, characterID, dhState),
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

	dhSnapshot, err := s.app.UpdateSnapshotState(ctx, campaignID, in)
	if err != nil {
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
