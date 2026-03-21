package charactertransport

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListCharacters returns a page of character records for a campaign.
func (s *Service) ListCharacters(ctx context.Context, in *campaignv1.ListCharactersRequest) (*campaignv1.ListCharactersResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list characters request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	page, err := s.app.ListCharacters(ctx, campaignID, in.GetPageToken(), in.GetPageSize())
	if err != nil {
		return nil, err
	}

	response := &campaignv1.ListCharactersResponse{
		NextPageToken: page.nextPageToken,
	}
	if len(page.characters) == 0 {
		return response, nil
	}

	response.Characters = make([]*campaignv1.Character, 0, len(page.characters))
	for _, ch := range page.characters {
		response.Characters = append(response.Characters, CharacterToProto(ch))
	}

	return response, nil
}

// ListCharacterProfiles returns a page of character profiles for a campaign.
func (s *Service) ListCharacterProfiles(ctx context.Context, in *campaignv1.ListCharacterProfilesRequest) (*campaignv1.ListCharacterProfilesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list character profiles request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	response := &campaignv1.ListCharacterProfilesResponse{}
	page, systemID, err := s.app.ListCharacterProfiles(ctx, campaignID, in.GetPageToken(), in.GetPageSize())
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(string(systemID), string(bridge.SystemIDDaggerheart)) {
		return response, nil
	}

	response.NextPageToken = page.nextPageToken
	if len(page.profiles) == 0 {
		return response, nil
	}

	response.Profiles = make([]*campaignv1.CharacterProfile, 0, len(page.profiles))
	for _, profile := range page.profiles {
		response.Profiles = append(response.Profiles, DaggerheartProfileToProto(campaignID, profile.CharacterID, profile, s.app.stores.DaggerheartContent))
	}

	return response, nil
}

// GetCharacterSheet returns a character sheet (character, profile, and state).
func (s *Service) GetCharacterSheet(ctx context.Context, in *campaignv1.GetCharacterSheetRequest) (*campaignv1.GetCharacterSheetResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get character sheet request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}

	sheet, err := s.app.GetCharacterSheet(ctx, campaignID, characterID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.GetCharacterSheetResponse{
		Character: CharacterToProto(sheet.character),
		Profile:   DaggerheartProfileToProto(campaignID, characterID, sheet.profile, s.app.stores.DaggerheartContent),
		State:     DaggerheartStateToProto(campaignID, characterID, sheet.state),
	}, nil
}
