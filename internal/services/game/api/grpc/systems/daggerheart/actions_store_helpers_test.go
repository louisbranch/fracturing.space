package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/test/mock/gamefakes"
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
