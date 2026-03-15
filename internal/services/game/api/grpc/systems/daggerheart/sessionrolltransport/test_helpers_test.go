package sessionrolltransport

import (
	"context"

	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type testCampaignStore struct {
	record storage.CampaignRecord
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	return s.record, nil
}

type testSessionStore struct {
	record storage.SessionRecord
}

func (s testSessionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	return s.record, nil
}

type testSessionGateStore struct{}

func (testSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type testDaggerheartStore struct {
	state projectionstore.DaggerheartCharacterState
}

func (s testDaggerheartStore) GetDaggerheartCharacterState(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
	return s.state, nil
}

type testEventStore struct {
	latestSeq uint64
}

func (s testEventStore) GetLatestEventSeq(context.Context, string) (uint64, error) {
	return s.latestSeq, nil
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
		deps.SessionGate = testSessionGateStore{}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = testDaggerheartStore{state: projectionstore.DaggerheartCharacterState{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Hope:        3,
		}}
	}
	if deps.Event == nil {
		deps.Event = testEventStore{latestSeq: 4}
	}
	if deps.SeedFunc == nil {
		deps.SeedFunc = func() (int64, error) { return 42, nil }
	}
	return NewHandler(deps)
}
