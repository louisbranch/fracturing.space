package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// CreateCharacter creates a character (PC/NPC/etc) for a campaign.
func (s *CharacterService) CreateCharacter(ctx context.Context, in *campaignv1.CreateCharacterRequest) (*campaignv1.CreateCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create character request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	created, err := newCharacterApplication(s).CreateCharacter(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.CreateCharacterResponse{Character: characterToProto(created)}, nil
}

// UpdateCharacter updates a character's metadata.
func (s *CharacterService) UpdateCharacter(ctx context.Context, in *campaignv1.UpdateCharacterRequest) (*campaignv1.UpdateCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update character request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	updated, err := newCharacterApplication(s).UpdateCharacter(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.UpdateCharacterResponse{Character: characterToProto(updated)}, nil
}

// DeleteCharacter deletes a character.
func (s *CharacterService) DeleteCharacter(ctx context.Context, in *campaignv1.DeleteCharacterRequest) (*campaignv1.DeleteCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete character request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	ch, err := newCharacterApplication(s).DeleteCharacter(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.DeleteCharacterResponse{Character: characterToProto(ch)}, nil
}

// SetDefaultControl assigns a campaign-scoped default controller for a character.
func (s *CharacterService) SetDefaultControl(ctx context.Context, in *campaignv1.SetDefaultControlRequest) (*campaignv1.SetDefaultControlResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set default control request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	characterID, participantID, err := newCharacterApplication(s).SetDefaultControl(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
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

// PatchCharacterProfile patches a character profile (all fields optional).
func (s *CharacterService) PatchCharacterProfile(ctx context.Context, in *campaignv1.PatchCharacterProfileRequest) (*campaignv1.PatchCharacterProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "patch character profile request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	characterID, dhProfile, err := newCharacterApplication(s).PatchCharacterProfile(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.PatchCharacterProfileResponse{
		Profile: daggerheartProfileToProto(campaignID, characterID, dhProfile),
	}, nil
}
