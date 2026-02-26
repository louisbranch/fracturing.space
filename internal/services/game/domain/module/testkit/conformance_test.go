package testkit

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type fakeDaggerheartStore struct {
	storage.DaggerheartStore
}

func TestValidateSystemConformance_Daggerheart(t *testing.T) {
	mod := daggerheart.NewModule()
	adapter := daggerheart.NewAdapter(fakeDaggerheartStore{})
	ValidateSystemConformance(t, mod, adapter)
}

func TestValidateAdapterIdempotency_Daggerheart(t *testing.T) {
	store := newMemDaggerheartStore()
	adapter := daggerheart.NewAdapter(store)
	ValidateAdapterIdempotency(t, adapter)
}

// memDaggerheartStore is a minimal in-memory implementation of
// storage.DaggerheartStore sufficient for adapter idempotency testing.
type memDaggerheartStore struct {
	mu          sync.Mutex
	profiles    map[string]storage.DaggerheartCharacterProfile
	states      map[string]storage.DaggerheartCharacterState
	snapshots   map[string]storage.DaggerheartSnapshot
	countdowns  map[string]storage.DaggerheartCountdown
	adversaries map[string]storage.DaggerheartAdversary
}

func newMemDaggerheartStore() *memDaggerheartStore {
	return &memDaggerheartStore{
		profiles:    make(map[string]storage.DaggerheartCharacterProfile),
		states:      make(map[string]storage.DaggerheartCharacterState),
		snapshots:   make(map[string]storage.DaggerheartSnapshot),
		countdowns:  make(map[string]storage.DaggerheartCountdown),
		adversaries: make(map[string]storage.DaggerheartAdversary),
	}
}

func (m *memDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, p storage.DaggerheartCharacterProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[p.CampaignID+"/"+p.CharacterID] = p
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[campaignID+"/"+characterID]
	if !ok {
		return storage.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return p, nil
}

func (m *memDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, s storage.DaggerheartCharacterState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[s.CampaignID+"/"+s.CharacterID] = s
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.states[campaignID+"/"+characterID]
	if !ok {
		return storage.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return s, nil
}

func (m *memDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, s storage.DaggerheartSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[s.CampaignID] = s
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.snapshots[campaignID]
	if !ok {
		return storage.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return s, nil
}

func (m *memDaggerheartStore) PutDaggerheartCountdown(_ context.Context, c storage.DaggerheartCountdown) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.countdowns[c.CampaignID+"/"+c.CountdownID] = c
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.countdowns[campaignID+"/"+countdownID]
	if !ok {
		return storage.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return c, nil
}

func (m *memDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]storage.DaggerheartCountdown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []storage.DaggerheartCountdown
	prefix := campaignID + "/"
	for k, v := range m.countdowns {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *memDaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.countdowns, campaignID+"/"+countdownID)
	return nil
}

func (m *memDaggerheartStore) PutDaggerheartAdversary(_ context.Context, a storage.DaggerheartAdversary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.adversaries[a.CampaignID+"/"+a.AdversaryID] = a
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.adversaries[campaignID+"/"+adversaryID]
	if !ok {
		return storage.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return a, nil
}

func (m *memDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, _ string) ([]storage.DaggerheartAdversary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []storage.DaggerheartAdversary
	prefix := campaignID + "/"
	for k, v := range m.adversaries {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *memDaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.adversaries, campaignID+"/"+adversaryID)
	return nil
}

// Reset clears all stored data between test phases.
func (m *memDaggerheartStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.profiles)
	clear(m.states)
	clear(m.snapshots)
	clear(m.countdowns)
	clear(m.adversaries)
}

var _ storage.DaggerheartStore = (*memDaggerheartStore)(nil)

// storeResetter allows ValidateAdapterIdempotency to clear store state
// between event types without depending on system-specific store types.
var _ fmt.Stringer = (*memDaggerheartStore)(nil) // compile guard only

func (m *memDaggerheartStore) String() string { return "memDaggerheartStore" }
