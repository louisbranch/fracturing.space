package gmmovetransport

import (
	"context"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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

type testSessionStore struct {
	record storage.SessionRecord
	err    error
}

func (s testSessionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	if s.err != nil {
		return storage.SessionRecord{}, s.err
	}
	return s.record, nil
}

type testDaggerheartStore struct {
	snapshot projectionstore.DaggerheartSnapshot
	err      error
}

func (s testDaggerheartStore) GetDaggerheartSnapshot(context.Context, string) (projectionstore.DaggerheartSnapshot, error) {
	if s.err != nil {
		return projectionstore.DaggerheartSnapshot{}, s.err
	}
	return s.snapshot, nil
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.Session == nil {
		deps.Session = testSessionStore{record: storage.SessionRecord{
			ID:         "sess-1",
			CampaignID: "camp-1",
			Status:     session.StatusActive,
		}}
	}
	if deps.SessionGate == nil {
		deps.SessionGate = testGateStore{err: storage.ErrNotFound}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = testDaggerheartStore{
			snapshot: projectionstore.DaggerheartSnapshot{
				CampaignID: "camp-1",
				GMFear:     0,
			},
		}
	}
	return NewHandler(deps)
}

func testContext() context.Context {
	ctx := grpcmeta.WithRequestID(context.Background(), "req-1")
	return grpcmeta.WithInvocationID(ctx, "inv-1")
}
