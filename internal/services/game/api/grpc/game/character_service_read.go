package game

import (
	"context"
	"errors"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
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
		return nil, status.Errorf(codes.Internal, "list characters: %v", err)
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
		return nil, status.Errorf(codes.Internal, "get daggerheart profile: %v", err)
	}

	dhState, err := s.stores.SystemStores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "get daggerheart state: %v", err)
	}

	return &campaignv1.GetCharacterSheetResponse{
		Character: characterToProto(ch),
		Profile:   daggerheartProfileToProto(campaignID, characterID, dhProfile),
		State:     daggerheartStateToProto(campaignID, characterID, dhState),
	}, nil
}
