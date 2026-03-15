package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/charactertransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// CreateCharacter creates a character (PC/NPC/etc) for a campaign.
func (s *CharacterService) CreateCharacter(ctx context.Context, in *campaignv1.CreateCharacterRequest) (*campaignv1.CreateCharacterResponse, error) {
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

	return &campaignv1.CreateCharacterResponse{Character: charactertransport.CharacterToProto(created)}, nil
}

// UpdateCharacter updates a character's metadata.
func (s *CharacterService) UpdateCharacter(ctx context.Context, in *campaignv1.UpdateCharacterRequest) (*campaignv1.UpdateCharacterResponse, error) {
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

	return &campaignv1.UpdateCharacterResponse{Character: charactertransport.CharacterToProto(updated)}, nil
}

// DeleteCharacter deletes a character.
func (s *CharacterService) DeleteCharacter(ctx context.Context, in *campaignv1.DeleteCharacterRequest) (*campaignv1.DeleteCharacterResponse, error) {
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

	return &campaignv1.DeleteCharacterResponse{Character: charactertransport.CharacterToProto(ch)}, nil
}

// SetDefaultControl assigns a campaign-scoped default controller for a character.
func (s *CharacterService) SetDefaultControl(ctx context.Context, in *campaignv1.SetDefaultControlRequest) (*campaignv1.SetDefaultControlResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set default control request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	characterID, participantID, err := s.app.SetDefaultControl(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	var participantIDValue *wrapperspb.StringValue
	if participantID != "" {
		participantIDValue = wrapperspb.String(participantID)
	}
	return &campaignv1.SetDefaultControlResponse{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		ParticipantId: participantIDValue,
	}, nil
}

// ClaimCharacterControl claims control of an unassigned character for the current participant.
func (s *CharacterService) ClaimCharacterControl(ctx context.Context, in *campaignv1.ClaimCharacterControlRequest) (*campaignv1.ClaimCharacterControlResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "claim character control request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	characterID, participantID, err := s.app.ClaimCharacterControl(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.ClaimCharacterControlResponse{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		ParticipantId: wrapperspb.String(participantID),
	}, nil
}

// ReleaseCharacterControl releases control of a character currently controlled by the current participant.
func (s *CharacterService) ReleaseCharacterControl(ctx context.Context, in *campaignv1.ReleaseCharacterControlRequest) (*campaignv1.ReleaseCharacterControlResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "release character control request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	characterID, err := s.app.ReleaseCharacterControl(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.ReleaseCharacterControlResponse{
		CampaignId:  campaignID,
		CharacterId: characterID,
	}, nil
}

// PatchCharacterProfile patches a character profile (all fields optional).
func (s *CharacterService) PatchCharacterProfile(ctx context.Context, in *campaignv1.PatchCharacterProfileRequest) (*campaignv1.PatchCharacterProfileResponse, error) {
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
		Profile: charactertransport.DaggerheartProfileToProto(campaignID, characterID, dhProfile),
	}, nil
}
