package charactertransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type characterListPage struct {
	characters    []storage.CharacterRecord
	nextPageToken string
}

type characterProfileListPage struct {
	profiles      []projectionstore.DaggerheartCharacterProfile
	nextPageToken string
}

type characterSheetState struct {
	character storage.CharacterRecord
	profile   projectionstore.DaggerheartCharacterProfile
	state     projectionstore.DaggerheartCharacterState
}

func (c characterApplication) ListCharacters(ctx context.Context, campaignID, pageToken string, pageSize int32) (characterListPage, error) {
	campaignRecord, err := c.loadReadableCampaign(ctx, campaignID)
	if err != nil {
		return characterListPage{}, err
	}

	resolvedPageSize := pagination.ClampPageSize(pageSize, pagination.PageSizeConfig{
		Default: defaultListCharactersPageSize,
		Max:     maxListCharactersPageSize,
	})
	page, err := c.stores.Character.ListCharacters(ctx, campaignRecord.ID, resolvedPageSize, pageToken)
	if err != nil {
		return characterListPage{}, grpcerror.Internal("list characters", err)
	}
	return characterListPage{
		characters:    page.Characters,
		nextPageToken: page.NextPageToken,
	}, nil
}

func (c characterApplication) ListCharacterProfiles(ctx context.Context, campaignID, pageToken string, pageSize int32) (characterProfileListPage, bridge.SystemID, error) {
	campaignRecord, err := c.loadReadableCampaign(ctx, campaignID)
	if err != nil {
		return characterProfileListPage{}, "", err
	}
	systemID := handler.SystemIDFromCampaignRecord(campaignRecord)
	if !strings.EqualFold(string(systemID), string(bridge.SystemIDDaggerheart)) {
		return characterProfileListPage{}, systemID, nil
	}

	resolvedPageSize := pagination.ClampPageSize(pageSize, pagination.PageSizeConfig{
		Default: defaultListCharactersPageSize,
		Max:     maxListCharactersPageSize,
	})
	page, err := c.stores.Daggerheart.ListDaggerheartCharacterProfiles(ctx, campaignRecord.ID, resolvedPageSize, pageToken)
	if err != nil {
		return characterProfileListPage{}, systemID, grpcerror.Internal("list daggerheart character profiles", err)
	}
	return characterProfileListPage{
		profiles:      page.Profiles,
		nextPageToken: page.NextPageToken,
	}, systemID, nil
}

func (c characterApplication) GetCharacterSheet(ctx context.Context, campaignID, characterID string) (characterSheetState, error) {
	campaignRecord, err := c.loadReadableCampaign(ctx, campaignID)
	if err != nil {
		return characterSheetState{}, err
	}

	ch, err := c.stores.Character.GetCharacter(ctx, campaignRecord.ID, characterID)
	if err != nil {
		return characterSheetState{}, err
	}

	dhProfile, err := c.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignRecord.ID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return characterSheetState{}, grpcerror.Internal("get daggerheart profile", err)
	}

	dhState, err := c.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignRecord.ID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return characterSheetState{}, grpcerror.Internal("get daggerheart state", err)
	}

	return characterSheetState{
		character: ch,
		profile:   dhProfile,
		state:     dhState,
	}, nil
}

func (c characterApplication) loadReadableCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := authz.RequireReadPolicy(ctx, c.auth, campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}
	return campaignRecord, nil
}
