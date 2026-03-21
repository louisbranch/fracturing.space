package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/metadata"
)

func contextWithSessionID(sessionID string) context.Context {
	md := metadata.Pairs(grpcmeta.SessionIDHeader, sessionID)
	return metadata.NewIncomingContext(context.Background(), md)
}

func optionalInt(value int) *int {
	return &value
}

func configureNoopDomain(svc *DaggerheartService) {
	svc.stores.Write.Executor = &fakeDomainEngine{}
}

func configureActionRollDomain(t *testing.T, svc *DaggerheartService, requestID string) {
	t.Helper()
	eventStore := svc.stores.Event.(*fakeEventStore)
	payloadJSON, err := json.Marshal(map[string]string{"request_id": requestID})
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	svc.stores.Write.Executor = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   testTimestamp,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   requestID,
				EntityType:  "roll",
				EntityID:    requestID,
				PayloadJSON: payloadJSON,
			}),
		},
	}}
}

func newActionTestService() *DaggerheartService {
	campaignStore := newFakeCampaignStore()
	campaignStore.Campaigns["camp-1"] = storage.CampaignRecord{
		ID:     "camp-1",
		Status: campaign.StatusActive,
		System: bridge.SystemIDDaggerheart,
	}

	dhStore := newFakeDaggerheartStore()
	dhStore.Profiles["camp-1:char-1"] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HpMax:       6,
		StressMax:   6,
		ArmorMax:    2,
	}
	dhStore.States["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     mechanics.HopeMax,
		Stress:      3,
		Armor:       0,
		LifeState:   daggerheartstate.LifeStateAlive,
	}
	dhStore.Profiles["camp-1:char-2"] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-2",
		HpMax:       8,
		StressMax:   6,
		ArmorMax:    1,
	}
	dhStore.States["camp-1:char-2"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-2",
		Hp:          8,
		Hope:        3,
		HopeMax:     mechanics.HopeMax,
		Stress:      1,
		Armor:       0,
		LifeState:   daggerheartstate.LifeStateAlive,
	}

	sessStore := newFakeSessionStore()
	sessStore.Sessions["camp-1:sess-1"] = storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Status:     session.StatusActive,
	}

	return &DaggerheartService{
		stores: Stores{
			Campaign:         campaignStore,
			Content:          newFakeContentStore(),
			Daggerheart:      dhStore,
			Character:        newFakeCharacterStore(),
			Event:            newFakeActionEventStore(),
			SessionGate:      &fakeSessionGateStore{},
			SessionSpotlight: &fakeSessionSpotlightStore{},
			Session:          sessStore,
			Write:            domainwriteexec.WritePath{Executor: &fakeDomainEngine{}, Runtime: testRuntime},
		},
		seedFunc: func() (int64, error) { return 42, nil },
	}
}
