package snapshottransport

import (
	"context"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type snapshotReadState struct {
	systemState     projectionstore.DaggerheartSnapshot
	characterStates []projectionstore.DaggerheartCharacterState
}

func (a snapshotApplication) GetSnapshot(ctx context.Context, campaignID string) (snapshotReadState, error) {
	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return snapshotReadState{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return snapshotReadState{}, err
	}
	if err := authz.RequireReadPolicy(ctx, a.auth, campaignRecord); err != nil {
		return snapshotReadState{}, err
	}

	systemState, err := a.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return snapshotReadState{}, grpcerror.Internal("get daggerheart snapshot", err)
		}
		systemState = projectionstore.DaggerheartSnapshot{CampaignID: campaignID}
	}

	charPage, err := a.stores.Character.ListCharacters(ctx, campaignID, 100, "")
	if err != nil {
		return snapshotReadState{}, grpcerror.Internal("list characters", err)
	}

	characterStates := make([]projectionstore.DaggerheartCharacterState, 0, len(charPage.Characters))
	for _, record := range charPage.Characters {
		state, err := a.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, record.ID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return snapshotReadState{}, grpcerror.Internal("get daggerheart character state", err)
		}
		characterStates = append(characterStates, state)
	}

	return snapshotReadState{
		systemState:     systemState,
		characterStates: characterStates,
	}, nil
}
