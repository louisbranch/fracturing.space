package gmmovetransport

import (
	"context"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmconsequence"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
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

func (s testDaggerheartStore) GetDaggerheartAdversary(context.Context, string, string) (projectionstore.DaggerheartAdversary, error) {
	return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
}

func (s testDaggerheartStore) GetDaggerheartEnvironmentEntity(context.Context, string, string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
}

type testSpotlightStore struct {
	spotlight storage.SessionSpotlight
	err       error
}

func (s testSpotlightStore) GetSessionSpotlight(context.Context, string, string) (storage.SessionSpotlight, error) {
	if s.err != nil {
		return storage.SessionSpotlight{}, s.err
	}
	return s.spotlight, nil
}

type testContentStore struct {
	adversaryEntry contentstore.DaggerheartAdversaryEntry
	environment    contentstore.DaggerheartEnvironment
	err            error
}

func (s testContentStore) GetDaggerheartAdversaryEntry(context.Context, string) (contentstore.DaggerheartAdversaryEntry, error) {
	if s.err != nil {
		return contentstore.DaggerheartAdversaryEntry{}, s.err
	}
	return s.adversaryEntry, nil
}

func (s testContentStore) GetDaggerheartEnvironment(context.Context, string) (contentstore.DaggerheartEnvironment, error) {
	if s.err != nil {
		return contentstore.DaggerheartEnvironment{}, s.err
	}
	return s.environment, nil
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
	if deps.SessionSpotlight == nil {
		deps.SessionSpotlight = testSpotlightStore{err: storage.ErrNotFound}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = testDaggerheartStore{
			snapshot: projectionstore.DaggerheartSnapshot{
				CampaignID: "camp-1",
				GMFear:     0,
			},
		}
	}
	if deps.Content == nil {
		deps.Content = testContentStore{}
	}
	if deps.ExecuteCoreCommand == nil {
		deps.ExecuteCoreCommand = func(context.Context, gmconsequence.CoreCommandInput) error { return nil }
	}
	return NewHandler(deps)
}

func testContext() context.Context {
	ctx := grpcmeta.WithRequestID(context.Background(), "req-1")
	return grpcmeta.WithInvocationID(ctx, "inv-1")
}
