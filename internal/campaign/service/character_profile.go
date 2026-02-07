package service

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	"github.com/louisbranch/fracturing.space/internal/campaign/domain"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetCharacterSheet returns a character sheet (character, profile, and state).
func (s *CampaignService) GetCharacterSheet(ctx context.Context, in *campaignv1.GetCharacterSheetRequest) (*campaignv1.GetCharacterSheetResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get character sheet request is required")
	}

	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}
	if s.stores.CharacterProfile == nil {
		return nil, status.Error(codes.Internal, "character profile store is not configured")
	}
	if s.stores.CharacterState == nil {
		return nil, status.Error(codes.Internal, "character state store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	character, err := s.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "character not found")
		}
		return nil, status.Errorf(codes.Internal, "get character: %v", err)
	}

	profile, err := s.stores.CharacterProfile.GetCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "character profile not found")
		}
		return nil, status.Errorf(codes.Internal, "get character profile: %v", err)
	}

	state, err := s.stores.CharacterState.GetCharacterState(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "character state not found")
		}
		return nil, status.Errorf(codes.Internal, "get character state: %v", err)
	}

	response := &campaignv1.GetCharacterSheetResponse{
		Character: characterToProto(character),
		Profile:   characterProfileToProto(profile),
		State:     characterStateToProto(state),
	}

	return response, nil
}

// PatchCharacterProfile patches a character profile (all fields optional).
func (s *CampaignService) PatchCharacterProfile(ctx context.Context, in *campaignv1.PatchCharacterProfileRequest) (*campaignv1.PatchCharacterProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "patch character profile request is required")
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

	existing, err := s.stores.CharacterProfile.GetCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "character profile not found")
		}
		return nil, status.Errorf(codes.Internal, "get character profile: %v", err)
	}

	patch := domain.PatchCharacterProfileInput{}
	if len(in.Traits) > 0 {
		traits := make(map[string]int)
		for k, v := range in.Traits {
			traits[k] = int(v)
		}
		patch.Traits = traits
	}
	if in.HpMax != nil {
		hpMax := int(*in.HpMax)
		patch.HpMax = &hpMax
	}
	if in.StressMax != nil {
		stressMax := int(*in.StressMax)
		patch.StressMax = &stressMax
	}
	if in.Evasion != nil {
		evasion := int(*in.Evasion)
		patch.Evasion = &evasion
	}
	if in.MajorThreshold != nil {
		majorThreshold := int(*in.MajorThreshold)
		patch.MajorThreshold = &majorThreshold
	}
	if in.SevereThreshold != nil {
		severeThreshold := int(*in.SevereThreshold)
		patch.SevereThreshold = &severeThreshold
	}

	updated, err := domain.PatchCharacterProfile(existing, patch)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidProfileHpMax) ||
			errors.Is(err, domain.ErrInvalidProfileStressMax) ||
			errors.Is(err, domain.ErrInvalidProfileEvasion) ||
			errors.Is(err, domain.ErrInvalidProfileThresholds) ||
			errors.Is(err, domain.ErrInvalidTraitValue) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "patch character profile: %v", err)
	}

	if err := s.stores.CharacterProfile.PutCharacterProfile(ctx, updated); err != nil {
		return nil, status.Errorf(codes.Internal, "persist character profile: %v", err)
	}

	response := &campaignv1.PatchCharacterProfileResponse{
		Profile: characterProfileToProto(updated),
	}

	return response, nil
}

// characterProfileToProto converts a domain CharacterProfile to protobuf.
func characterProfileToProto(profile domain.CharacterProfile) *campaignv1.CharacterProfile {
	traits := make(map[string]int32)
	for k, v := range profile.Traits {
		traits[k] = int32(v)
	}

	return &campaignv1.CharacterProfile{
		CampaignId:      profile.CampaignID,
		CharacterId:     profile.CharacterID,
		Traits:          traits,
		HpMax:           int32(profile.HpMax),
		StressMax:       int32(profile.StressMax),
		Evasion:         int32(profile.Evasion),
		MajorThreshold:  int32(profile.MajorThreshold),
		SevereThreshold: int32(profile.SevereThreshold),
	}
}

// characterToProto converts a domain Character to protobuf.
func characterToProto(character domain.Character) *campaignv1.Character {
	return &campaignv1.Character{
		Id:         character.ID,
		CampaignId: character.CampaignID,
		Name:       character.Name,
		Kind:       characterKindToProto(character.Kind),
		Notes:      character.Notes,
		CreatedAt:  timestamppb.New(character.CreatedAt),
		UpdatedAt:  timestamppb.New(character.UpdatedAt),
	}
}
