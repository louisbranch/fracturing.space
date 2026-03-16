package adversarytransport

import (
	"context"

	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	gmetadata "google.golang.org/grpc/metadata"
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
	err error
}

func (s testSessionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	if s.err != nil {
		return storage.SessionRecord{}, s.err
	}
	return storage.SessionRecord{}, nil
}

type testGateStore struct {
	gate storage.SessionGate
	err  error
}

func (s testGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	if s.err != nil {
		return storage.SessionGate{}, s.err
	}
	if s.gate.GateID != "" {
		return s.gate, nil
	}
	return storage.SessionGate{}, storage.ErrNotFound
}

type testDaggerheartStore struct {
	adversaries map[string]projectionstore.DaggerheartAdversary
	err         error
}

func (s *testDaggerheartStore) GetDaggerheartAdversary(_ context.Context, _, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if s.err != nil {
		return projectionstore.DaggerheartAdversary{}, s.err
	}
	adversary, ok := s.adversaries[adversaryID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (s *testDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, _, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([]projectionstore.DaggerheartAdversary, 0, len(s.adversaries))
	for _, adversary := range s.adversaries {
		if sessionID == "" || adversary.SessionID == sessionID {
			out = append(out, adversary)
		}
	}
	return out, nil
}

func testContext() context.Context {
	return gmetadata.NewIncomingContext(context.Background(), gmetadata.Pairs("x-session-id", "sess-1"))
}

type testContentStore struct {
	adversaryEntries map[string]contentstore.DaggerheartAdversaryEntry
}

func (s testContentStore) GetDaggerheartAdversaryEntry(_ context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error) {
	entry, ok := s.adversaryEntries[id]
	if !ok {
		return contentstore.DaggerheartAdversaryEntry{}, storage.ErrNotFound
	}
	return entry, nil
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
		deps.Session = testSessionStore{}
	}
	if deps.Gate == nil {
		deps.Gate = testGateStore{err: storage.ErrNotFound}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{}}
	}
	if deps.Content == nil {
		deps.Content = testContentStore{adversaryEntries: map[string]contentstore.DaggerheartAdversaryEntry{
			"adversary.rival": {ID: "adversary.rival", Name: "Rival", HP: 6, Stress: 2, Difficulty: 11, MajorThreshold: 4, SevereThreshold: 7, Armor: 1},
		}}
	}
	return NewHandler(deps)
}
