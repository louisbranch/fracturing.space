package outcometransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/test/mock/gamefakes"
)

type fakeSessionGateStore struct{}

func (s *fakeSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type fakeSessionSpotlightStore struct{}

func (s *fakeSessionSpotlightStore) GetSessionSpotlight(context.Context, string, string) (storage.SessionSpotlight, error) {
	return storage.SessionSpotlight{}, storage.ErrNotFound
}

type fakeContentStore struct {
	subclasses map[string]contentstore.DaggerheartSubclass
}

func (s *fakeContentStore) GetDaggerheartSubclass(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	sc, ok := s.subclasses[id]
	if !ok {
		return contentstore.DaggerheartSubclass{}, storage.ErrNotFound
	}
	return sc, nil
}

type callbackRecorder struct {
	systemCommands []SystemCommandInput
	coreCommands   []CoreCommandInput
	stressCalls    []ApplyStressVulnerableConditionInput
}

func newTestHandler() (*Handler, *gamefakes.EventStore, *callbackRecorder) {
	recorder := &callbackRecorder{}
	campaigns := gamefakes.NewCampaignStore()
	campaigns.Campaigns["camp-1"] = storage.CampaignRecord{
		ID:     "camp-1",
		Status: campaign.StatusActive,
		System: systembridge.SystemIDDaggerheart,
	}

	sessions := gamefakes.NewSessionStore()
	sessions.Sessions["camp-1:sess-1"] = storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Status:     session.StatusActive,
	}

	daggerheartStore := gamefakes.NewDaggerheartStore()
	daggerheartStore.Profiles["camp-1:char-1"] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		StressMax:   6,
	}
	daggerheartStore.States["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hope:        2,
		HopeMax:     bridge.HopeMax,
		Stress:      3,
		Hp:          6,
	}
	daggerheartStore.Snapshots["camp-1"] = projectionstore.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     2,
	}

	events := gamefakes.NewEventStore()

	return NewHandler(Dependencies{
		Campaign:         campaigns,
		Session:          sessions,
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      daggerheartStore,
		Content:          &fakeContentStore{subclasses: make(map[string]contentstore.DaggerheartSubclass)},
		Event:            events,
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			recorder.systemCommands = append(recorder.systemCommands, in)
			return nil
		},
		ExecuteCoreCommand: func(_ context.Context, in CoreCommandInput) error {
			recorder.coreCommands = append(recorder.coreCommands, in)
			return nil
		},
		ApplyStressVulnerableCondition: func(_ context.Context, in ApplyStressVulnerableConditionInput) error {
			recorder.stressCalls = append(recorder.stressCalls, in)
			return nil
		},
	}), events, recorder
}
