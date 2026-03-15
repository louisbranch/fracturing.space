package charactermutationtransport

import (
	"context"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/metadata"
)

type testCampaignStore struct {
	record storage.CampaignRecord
	err    error
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	if s.err != nil {
		return storage.CampaignRecord{}, s.err
	}
	return s.record, nil
}

type testDaggerheartStore struct {
	profiles map[string]projectionstore.DaggerheartCharacterProfile
	getErr   error
}

func (s *testDaggerheartStore) GetDaggerheartCharacterProfile(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
	if s.getErr != nil {
		return projectionstore.DaggerheartCharacterProfile{}, s.getErr
	}
	for _, profile := range s.profiles {
		return profile, nil
	}
	return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
}

func testProfile(campaignID, characterID string) projectionstore.DaggerheartCharacterProfile {
	return projectionstore.DaggerheartCharacterProfile{
		CampaignID:   campaignID,
		CharacterID:  characterID,
		Level:        1,
		GoldHandfuls: 1,
		GoldBags:     2,
		GoldChests:   3,
	}
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = &testDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"camp-1:char-1": testProfile("camp-1", "char-1"),
			},
		}
	}
	return NewHandler(deps)
}

func testContext() context.Context {
	ctx := grpcmeta.WithRequestID(context.Background(), "req-1")
	ctx = grpcmeta.WithInvocationID(ctx, "inv-1")
	return metadata.NewIncomingContext(ctx, metadata.Pairs(grpcmeta.SessionIDHeader, "sess-1"))
}
