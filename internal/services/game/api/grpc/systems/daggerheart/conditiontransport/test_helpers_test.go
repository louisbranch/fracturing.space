package conditiontransport

import (
	"context"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/metadata"
)

type testCampaignStore struct {
	record storage.CampaignRecord
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	return s.record, nil
}

type testSessionGateStore struct{}

func (testSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type testDaggerheartStore struct {
	state     projectionstore.DaggerheartCharacterState
	adversary projectionstore.DaggerheartAdversary
}

func (s testDaggerheartStore) GetDaggerheartCharacterState(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
	return s.state, nil
}

func (s testDaggerheartStore) GetDaggerheartAdversary(context.Context, string, string) (projectionstore.DaggerheartAdversary, error) {
	return s.adversary, nil
}

type testEventStore struct {
	event event.Event
}

func (s testEventStore) GetEventBySeq(context.Context, string, uint64) (event.Event, error) {
	return s.event, nil
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.SessionGate == nil {
		deps.SessionGate = testSessionGateStore{}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = testDaggerheartStore{
			state: projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				LifeState:   daggerheart.LifeStateAlive,
				Conditions:  []projectionstore.DaggerheartConditionState{{Standard: daggerheart.ConditionHidden}},
			},
			adversary: projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				Conditions:  []projectionstore.DaggerheartConditionState{{Standard: daggerheart.ConditionHidden}},
			},
		}
	}
	if deps.Event == nil {
		deps.Event = testEventStore{}
	}
	return NewHandler(deps)
}

func testContextWithSessionID(sessionID string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.SessionIDHeader, sessionID))
}
