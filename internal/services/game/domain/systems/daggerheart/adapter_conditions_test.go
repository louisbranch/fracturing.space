package daggerheart

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type memoryDaggerheartStore struct {
	states      map[string]storage.DaggerheartCharacterState
	profiles    map[string]storage.DaggerheartCharacterProfile
	snaps       map[string]storage.DaggerheartSnapshot
	countdowns  map[string]storage.DaggerheartCountdown
	adversaries map[string]storage.DaggerheartAdversary
}

func newMemoryDaggerheartStore() *memoryDaggerheartStore {
	return &memoryDaggerheartStore{
		states:      make(map[string]storage.DaggerheartCharacterState),
		profiles:    make(map[string]storage.DaggerheartCharacterProfile),
		snaps:       make(map[string]storage.DaggerheartSnapshot),
		countdowns:  make(map[string]storage.DaggerheartCountdown),
		adversaries: make(map[string]storage.DaggerheartAdversary),
	}
}

func (m *memoryDaggerheartStore) PutDaggerheartCharacterProfile(ctx context.Context, profile storage.DaggerheartCharacterProfile) error {
	key := profile.CampaignID + ":" + profile.CharacterID
	m.profiles[key] = profile
	return nil
}

func (m *memoryDaggerheartStore) GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	key := campaignID + ":" + characterID
	profile, ok := m.profiles[key]
	if !ok {
		return storage.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return profile, nil
}

func (m *memoryDaggerheartStore) PutDaggerheartCharacterState(ctx context.Context, state storage.DaggerheartCharacterState) error {
	key := state.CampaignID + ":" + state.CharacterID
	m.states[key] = state
	return nil
}

func (m *memoryDaggerheartStore) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	key := campaignID + ":" + characterID
	state, ok := m.states[key]
	if !ok {
		return storage.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return state, nil
}

func (m *memoryDaggerheartStore) PutDaggerheartSnapshot(ctx context.Context, snap storage.DaggerheartSnapshot) error {
	m.snaps[snap.CampaignID] = snap
	return nil
}

func (m *memoryDaggerheartStore) GetDaggerheartSnapshot(ctx context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	snap, ok := m.snaps[campaignID]
	if !ok {
		return storage.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (m *memoryDaggerheartStore) PutDaggerheartCountdown(ctx context.Context, countdown storage.DaggerheartCountdown) error {
	key := countdown.CampaignID + ":" + countdown.CountdownID
	m.countdowns[key] = countdown
	return nil
}

func (m *memoryDaggerheartStore) GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	key := campaignID + ":" + countdownID
	countdown, ok := m.countdowns[key]
	if !ok {
		return storage.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (m *memoryDaggerheartStore) ListDaggerheartCountdowns(ctx context.Context, campaignID string) ([]storage.DaggerheartCountdown, error) {
	results := make([]storage.DaggerheartCountdown, 0)
	for _, countdown := range m.countdowns {
		if countdown.CampaignID == campaignID {
			results = append(results, countdown)
		}
	}
	return results, nil
}

func (m *memoryDaggerheartStore) DeleteDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) error {
	key := campaignID + ":" + countdownID
	if _, ok := m.countdowns[key]; !ok {
		return storage.ErrNotFound
	}
	delete(m.countdowns, key)
	return nil
}

func (m *memoryDaggerheartStore) PutDaggerheartAdversary(ctx context.Context, adversary storage.DaggerheartAdversary) error {
	key := adversary.CampaignID + ":" + adversary.AdversaryID
	m.adversaries[key] = adversary
	return nil
}

func (m *memoryDaggerheartStore) GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	key := campaignID + ":" + adversaryID
	adversary, ok := m.adversaries[key]
	if !ok {
		return storage.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (m *memoryDaggerheartStore) ListDaggerheartAdversaries(ctx context.Context, campaignID, sessionID string) ([]storage.DaggerheartAdversary, error) {
	results := make([]storage.DaggerheartAdversary, 0)
	for _, adversary := range m.adversaries {
		if adversary.CampaignID != campaignID {
			continue
		}
		if strings.TrimSpace(sessionID) != "" && adversary.SessionID != sessionID {
			continue
		}
		results = append(results, adversary)
	}
	return results, nil
}

func (m *memoryDaggerheartStore) DeleteDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) error {
	key := campaignID + ":" + adversaryID
	if _, ok := m.adversaries[key]; !ok {
		return storage.ErrNotFound
	}
	delete(m.adversaries, key)
	return nil
}

func TestAdapterApplyConditionChanged(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		Stress:      0,
		Armor:       0,
		Conditions:  []string{ConditionHidden},
	}

	payload := ConditionChangedPayload{
		CharacterID:     "char-1",
		ConditionsAfter: []string{ConditionVulnerable, ConditionHidden},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	adapter := NewAdapter(store)
	if err := adapter.ApplyEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Type:        EventTypeConditionChanged,
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("apply event: %v", err)
	}

	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if !ConditionsEqual(state.Conditions, []string{ConditionHidden, ConditionVulnerable}) {
		t.Fatalf("conditions = %v, want %v", state.Conditions, []string{ConditionHidden, ConditionVulnerable})
	}
}

func TestAdapterApplyConditionChangedRejectsUnknown(t *testing.T) {
	store := newMemoryDaggerheartStore()
	payload := ConditionChangedPayload{
		CharacterID:     "char-1",
		ConditionsAfter: []string{"mystery"},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	adapter := NewAdapter(store)
	if err := adapter.ApplyEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Type:        EventTypeConditionChanged,
		PayloadJSON: payloadJSON,
	}); err == nil {
		t.Fatal("expected error")
	}
}
