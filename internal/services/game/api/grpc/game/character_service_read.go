package game

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListCharacters returns a page of character records for a campaign.
func (s *CharacterService) ListCharacters(ctx context.Context, in *campaignv1.ListCharactersRequest) (*campaignv1.ListCharactersResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list characters request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requireReadPolicy(ctx, s.stores, c); err != nil {
		return nil, err
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListCharactersPageSize,
		Max:     maxListCharactersPageSize,
	})

	page, err := s.stores.Character.ListCharacters(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, grpcerror.Internal("list characters", err)
	}

	response := &campaignv1.ListCharactersResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Characters) == 0 {
		return response, nil
	}

	response.Characters = make([]*campaignv1.Character, 0, len(page.Characters))
	for _, ch := range page.Characters {
		response.Characters = append(response.Characters, characterToProto(ch))
	}

	return response, nil
}

// ListCharacterProfiles returns a page of character profiles for a campaign.
func (s *CharacterService) ListCharacterProfiles(ctx context.Context, in *campaignv1.ListCharacterProfilesRequest) (*campaignv1.ListCharacterProfilesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list character profiles request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requireReadPolicy(ctx, s.stores, c); err != nil {
		return nil, err
	}

	response := &campaignv1.ListCharacterProfilesResponse{}
	if !strings.EqualFold(string(c.System), string(bridge.SystemIDDaggerheart)) {
		return response, nil
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListCharactersPageSize,
		Max:     maxListCharactersPageSize,
	})

	page, err := s.stores.SystemStores.Daggerheart.ListDaggerheartCharacterProfiles(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, grpcerror.Internal("list daggerheart character profiles", err)
	}

	response.NextPageToken = page.NextPageToken
	if len(page.Profiles) == 0 {
		return response, nil
	}

	response.Profiles = make([]*campaignv1.CharacterProfile, 0, len(page.Profiles))
	for _, profile := range page.Profiles {
		response.Profiles = append(response.Profiles, daggerheartProfileToProto(campaignID, profile.CharacterID, profile))
	}

	return response, nil
}

// GetCharacterSheet returns a character sheet (character, profile, and state).
func (s *CharacterService) GetCharacterSheet(ctx context.Context, in *campaignv1.GetCharacterSheetRequest) (*campaignv1.GetCharacterSheetResponse, error) {
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

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requireReadPolicy(ctx, s.stores, c); err != nil {
		return nil, err
	}

	ch, err := s.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return nil, err
	}

	dhProfile, err := s.stores.SystemStores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, grpcerror.Internal("get daggerheart profile", err)
	}

	dhState, err := s.stores.SystemStores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, grpcerror.Internal("get daggerheart state", err)
	}

	return &campaignv1.GetCharacterSheetResponse{
		Character: characterToProto(ch),
		Profile:   daggerheartProfileToProto(campaignID, characterID, dhProfile),
		State:     daggerheartStateToProto(campaignID, characterID, dhState),
	}, nil
}
