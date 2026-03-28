package snapshottransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
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
		if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "get daggerheart snapshot"); lookupErr != nil {
			return snapshotReadState{}, lookupErr
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
			if grpcerror.OptionalLookupErrorContext(ctx, err, "get daggerheart character state") == nil {
				continue
			}
			return snapshotReadState{}, grpcerror.OptionalLookupErrorContext(ctx, err, "get daggerheart character state")
		}
		characterStates = append(characterStates, state)
	}

	return snapshotReadState{
		systemState:     systemState,
		characterStates: characterStates,
	}, nil
}
