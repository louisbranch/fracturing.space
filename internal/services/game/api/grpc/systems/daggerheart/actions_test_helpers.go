package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/testkit/gamefakes"
	"google.golang.org/grpc/metadata"
)

type fakeCampaignStore = gamefakes.CampaignStore
type fakeDaggerheartStore = gamefakes.DaggerheartStore
type fakeEventStore = gamefakes.EventStore

func newFakeCampaignStore() *fakeCampaignStore {
	return gamefakes.NewCampaignStore()
}

func newFakeDaggerheartStore() *fakeDaggerheartStore {
	return gamefakes.NewDaggerheartStore()
}

func newFakeActionEventStore() *fakeEventStore {
	return gamefakes.NewEventStore()
}

type fakeDomainEngine struct {
	store         storage.EventStore
	result        engine.Result
	resultsByType map[command.Type]engine.Result
	calls         int
	lastCommand   command.Command
	commands      []command.Command
}

func (f *fakeDomainEngine) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	f.calls++
	f.lastCommand = cmd
	f.commands = append(f.commands, cmd)

	result := f.result
	if len(f.resultsByType) > 0 {
		if selected, ok := f.resultsByType[cmd.Type]; ok {
			result = selected
		}
	}
	if f.store == nil {
		return result, nil
	}
	if len(result.Decision.Events) == 0 {
		return result, nil
	}
	stored := make([]event.Event, 0, len(result.Decision.Events))
	for _, evt := range result.Decision.Events {
		storedEvent, err := f.store.AppendEvent(ctx, evt)
		if err != nil {
			return engine.Result{}, err
		}
		stored = append(stored, storedEvent)
	}
	result.Decision.Events = stored
	return result, nil
}

type fakeCharacterStore = gamefakes.CharacterStore

func newFakeCharacterStore() *fakeCharacterStore {
	return gamefakes.NewCharacterStore()
}

type fakeSessionGateStore struct{}

func (s *fakeSessionGateStore) PutSessionGate(_ context.Context, _ storage.SessionGate) error {
	return nil
}

func (s *fakeSessionGateStore) GetSessionGate(_ context.Context, _, _, _ string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

func (s *fakeSessionGateStore) GetOpenSessionGate(_ context.Context, _, _ string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type fakeOpenSessionGateStore struct {
	gate storage.SessionGate
}

func (s *fakeOpenSessionGateStore) PutSessionGate(_ context.Context, _ storage.SessionGate) error {
	return nil
}

func (s *fakeOpenSessionGateStore) GetSessionGate(_ context.Context, _, _, _ string) (storage.SessionGate, error) {
	return s.gate, nil
}

func (s *fakeOpenSessionGateStore) GetOpenSessionGate(_ context.Context, _, _ string) (storage.SessionGate, error) {
	return s.gate, nil
}

type fakeSessionSpotlightStore struct{}

func (s *fakeSessionSpotlightStore) PutSessionSpotlight(_ context.Context, _ storage.SessionSpotlight) error {
	return nil
}

func (s *fakeSessionSpotlightStore) GetSessionSpotlight(_ context.Context, _, _ string) (storage.SessionSpotlight, error) {
	return storage.SessionSpotlight{}, storage.ErrNotFound
}

func (s *fakeSessionSpotlightStore) ClearSessionSpotlight(_ context.Context, _, _ string) error {
	return nil
}

type fakeSessionSpotlightStateStore struct {
	spotlight storage.SessionSpotlight
	exists    bool
}

func (s *fakeSessionSpotlightStateStore) PutSessionSpotlight(_ context.Context, spotlight storage.SessionSpotlight) error {
	s.spotlight = spotlight
	s.exists = true
	return nil
}

func (s *fakeSessionSpotlightStateStore) GetSessionSpotlight(_ context.Context, _, _ string) (storage.SessionSpotlight, error) {
	if !s.exists {
		return storage.SessionSpotlight{}, storage.ErrNotFound
	}
	return s.spotlight, nil
}

func (s *fakeSessionSpotlightStateStore) ClearSessionSpotlight(_ context.Context, _, _ string) error {
	s.exists = false
	s.spotlight = storage.SessionSpotlight{}
	return nil
}

type fakeSessionStore = gamefakes.SessionStore

func newFakeSessionStore() *fakeSessionStore {
	return gamefakes.NewSessionStore()
}

func contextWithSessionID(sessionID string) context.Context {
	md := metadata.Pairs(grpcmeta.SessionIDHeader, sessionID)
	return metadata.NewIncomingContext(context.Background(), md)
}

func optionalInt(value int) *int {
	return &value
}

func configureNoopDomain(svc *DaggerheartService) {
	svc.stores.Domain = &fakeDomainEngine{}
}

func configureActionRollDomain(t *testing.T, svc *DaggerheartService, requestID string) {
	t.Helper()
	eventStore := svc.stores.Event.(*fakeEventStore)
	payloadJSON, err := json.Marshal(map[string]string{"request_id": requestID})
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
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
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}

	dhStore := newFakeDaggerheartStore()
	dhStore.Profiles["camp-1:char-1"] = storage.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HpMax:       6,
		StressMax:   6,
		ArmorMax:    2,
	}
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     daggerheart.HopeMax,
		Stress:      3,
		Armor:       0,
		LifeState:   daggerheart.LifeStateAlive,
	}
	dhStore.Profiles["camp-1:char-2"] = storage.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-2",
		HpMax:       8,
		StressMax:   6,
		ArmorMax:    1,
	}
	dhStore.States["camp-1:char-2"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-2",
		Hp:          8,
		Hope:        3,
		HopeMax:     daggerheart.HopeMax,
		Stress:      1,
		Armor:       0,
		LifeState:   daggerheart.LifeStateAlive,
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
			Daggerheart:      dhStore,
			Character:        newFakeCharacterStore(),
			Event:            newFakeActionEventStore(),
			SessionGate:      &fakeSessionGateStore{},
			SessionSpotlight: &fakeSessionSpotlightStore{},
			Domain:           &fakeDomainEngine{},
			Session:          sessStore,
		},
		seedFunc: func() (int64, error) { return 42, nil },
	}
}
