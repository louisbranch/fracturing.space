package charactertransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateCharacter creates a character (PC/NPC/etc) for a campaign.
func (s *Service) CreateCharacter(ctx context.Context, in *campaignv1.CreateCharacterRequest) (*campaignv1.CreateCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create character request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	created, err := s.app.CreateCharacter(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.CreateCharacterResponse{Character: CharacterToProto(created)}, nil
}

// UpdateCharacter updates a character's metadata.
func (s *Service) UpdateCharacter(ctx context.Context, in *campaignv1.UpdateCharacterRequest) (*campaignv1.UpdateCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update character request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	updated, err := s.app.UpdateCharacter(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.UpdateCharacterResponse{Character: CharacterToProto(updated)}, nil
}

// DeleteCharacter deletes a character.
func (s *Service) DeleteCharacter(ctx context.Context, in *campaignv1.DeleteCharacterRequest) (*campaignv1.DeleteCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete character request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	ch, err := s.app.DeleteCharacter(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.DeleteCharacterResponse{Character: CharacterToProto(ch)}, nil
}

// PatchCharacterProfile patches a character profile (all fields optional).
func (s *Service) PatchCharacterProfile(ctx context.Context, in *campaignv1.PatchCharacterProfileRequest) (*campaignv1.PatchCharacterProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "patch character profile request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	characterID, dhProfile, err := s.app.PatchCharacterProfile(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.PatchCharacterProfileResponse{
		Profile: DaggerheartProfileToProto(campaignID, characterID, dhProfile, s.app.stores.DaggerheartContent),
	}, nil
}
