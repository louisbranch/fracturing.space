package damagetransport

import (
	"context"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
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

type testSessionGateStore struct{}

func (testSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type testOpenGateStore struct{}

func (testOpenGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{GateID: "gate-1"}, nil
}

type testDaggerheartStore struct {
	profile         projectionstore.DaggerheartCharacterProfile
	state           projectionstore.DaggerheartCharacterState
	adversary       projectionstore.DaggerheartAdversary
	listAdversaries []projectionstore.DaggerheartAdversary
	listErr         error
}

func (s testDaggerheartStore) GetDaggerheartCharacterProfile(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
	return s.profile, nil
}

func (s testDaggerheartStore) GetDaggerheartCharacterState(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
	return s.state, nil
}

func (s testDaggerheartStore) GetDaggerheartAdversary(context.Context, string, string) (projectionstore.DaggerheartAdversary, error) {
	return s.adversary, nil
}

func (s testDaggerheartStore) ListDaggerheartAdversaries(context.Context, string, string) ([]projectionstore.DaggerheartAdversary, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]projectionstore.DaggerheartAdversary(nil), s.listAdversaries...), nil
}

type testContentStore struct {
	adversaryEntries map[string]contentstore.DaggerheartAdversaryEntry
	armors           map[string]contentstore.DaggerheartArmor
	adversaryErr     error
	armorErr         error
}

func (s testContentStore) GetDaggerheartAdversaryEntry(_ context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error) {
	if s.adversaryErr != nil {
		return contentstore.DaggerheartAdversaryEntry{}, s.adversaryErr
	}
	entry, ok := s.adversaryEntries[id]
	if !ok {
		return contentstore.DaggerheartAdversaryEntry{}, storage.ErrNotFound
	}
	return entry, nil
}

func (s testContentStore) GetDaggerheartArmor(_ context.Context, id string) (contentstore.DaggerheartArmor, error) {
	if s.armorErr != nil {
		return contentstore.DaggerheartArmor{}, s.armorErr
	}
	a, ok := s.armors[id]
	if !ok {
		return contentstore.DaggerheartArmor{}, storage.ErrNotFound
	}
	return a, nil
}

type testEventStore struct {
	event event.Event
	err   error
}

func (s testEventStore) GetEventBySeq(context.Context, string, uint64) (event.Event, error) {
	if s.err != nil {
		return event.Event{}, s.err
	}
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
			profile: projectionstore.DaggerheartCharacterProfile{
				CampaignID:      "camp-1",
				CharacterID:     "char-1",
				MajorThreshold:  5,
				SevereThreshold: 8,
			},
			state: projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Hp:          10,
				Armor:       1,
			},
			adversary: projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				HP:          10,
				Armor:       1,
				Major:       5,
				Severe:      8,
			},
		}
	}
	if deps.Content == nil {
		deps.Content = testContentStore{
			adversaryEntries: map[string]contentstore.DaggerheartAdversaryEntry{
				"entry-goblin": {ID: "entry-goblin", Name: "Goblin", Role: "bruiser", HP: 10, Armor: 1, MajorThreshold: 5, SevereThreshold: 8},
			},
			armors: make(map[string]contentstore.DaggerheartArmor),
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
