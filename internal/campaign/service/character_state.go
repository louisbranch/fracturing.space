package service

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PatchCharacterState patches a character state (all fields optional).
func (s *CampaignService) PatchCharacterState(ctx context.Context, in *campaignv1.PatchCharacterStateRequest) (*campaignv1.PatchCharacterStateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "patch character state request is required")
	}

	if s.stores.CharacterState == nil {
		return nil, status.Error(codes.Internal, "character state store is not configured")
	}
	if s.stores.CharacterProfile == nil {
		return nil, status.Error(codes.Internal, "character profile store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	existing, err := s.stores.CharacterState.GetCharacterState(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "character state not found")
		}
		return nil, status.Errorf(codes.Internal, "get character state: %v", err)
	}

	profile, err := s.stores.CharacterProfile.GetCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "character profile not found")
		}
		return nil, status.Errorf(codes.Internal, "get character profile: %v", err)
	}

	patch := domain.PatchCharacterStateInput{}
	if in.Hope != nil {
		hope := int(*in.Hope)
		patch.Hope = &hope
	}
	if in.Stress != nil {
		stress := int(*in.Stress)
		patch.Stress = &stress
	}
	if in.Hp != nil {
		hp := int(*in.Hp)
		patch.Hp = &hp
	}

	updated, err := domain.PatchCharacterState(existing, patch, profile)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidHope) ||
			errors.Is(err, domain.ErrInvalidStress) ||
			errors.Is(err, domain.ErrInvalidHp) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "patch character state: %v", err)
	}

	if err := s.stores.CharacterState.PutCharacterState(ctx, updated); err != nil {
		return nil, status.Errorf(codes.Internal, "persist character state: %v", err)
	}

	response := &campaignv1.PatchCharacterStateResponse{
		State: characterStateToProto(updated),
	}

	return response, nil
}

// characterStateToProto converts a domain CharacterState to protobuf.
func characterStateToProto(state domain.CharacterState) *campaignv1.CharacterState {
	return &campaignv1.CharacterState{
		CampaignId:  state.CampaignID,
		CharacterId: state.CharacterID,
		Hope:        int32(state.Hope),
		Stress:      int32(state.Stress),
		Hp:          int32(state.Hp),
	}
}
