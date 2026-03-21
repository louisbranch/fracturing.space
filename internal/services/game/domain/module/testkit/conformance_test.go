package testkit

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestValidateSystemConformance_Daggerheart(t *testing.T) {
	mod := daggerheart.NewModule()
	adapter := daggerheart.NewAdapter(newMemDaggerheartStore())
	ValidateSystemConformance(t, mod, adapter)
}

func TestValidateAdapterIdempotency_Daggerheart(t *testing.T) {
	store := newMemDaggerheartStore()
	adapter := daggerheart.NewAdapter(store)
	ValidateAdapterIdempotency(t, adapter)
}

// memDaggerheartStore is a minimal in-memory implementation of
// projectionstore.Store sufficient for adapter idempotency testing.
type memDaggerheartStore struct {
	mu           sync.Mutex
	profiles     map[string]projectionstore.DaggerheartCharacterProfile
	states       map[string]projectionstore.DaggerheartCharacterState
	snapshots    map[string]projectionstore.DaggerheartSnapshot
	countdowns   map[string]projectionstore.DaggerheartCountdown
	adversaries  map[string]projectionstore.DaggerheartAdversary
	environments map[string]projectionstore.DaggerheartEnvironmentEntity
}

func newMemDaggerheartStore() *memDaggerheartStore {
	return &memDaggerheartStore{
		profiles:     make(map[string]projectionstore.DaggerheartCharacterProfile),
		states:       make(map[string]projectionstore.DaggerheartCharacterState),
		snapshots:    make(map[string]projectionstore.DaggerheartSnapshot),
		countdowns:   make(map[string]projectionstore.DaggerheartCountdown),
		adversaries:  make(map[string]projectionstore.DaggerheartAdversary),
		environments: make(map[string]projectionstore.DaggerheartEnvironmentEntity),
	}
}

func (m *memDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, p projectionstore.DaggerheartCharacterProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[p.CampaignID+"/"+p.CharacterID] = p
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[campaignID+"/"+characterID]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return p, nil
}

func (m *memDaggerheartStore) ListDaggerheartCharacterProfiles(_ context.Context, campaignID string, _ int, _ string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	page := projectionstore.DaggerheartCharacterProfilePage{
		Profiles: make([]projectionstore.DaggerheartCharacterProfile, 0),
	}
	prefix := campaignID + "/"
	for key, profile := range m.profiles {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			page.Profiles = append(page.Profiles, profile)
		}
	}
	return page, nil
}

func (m *memDaggerheartStore) DeleteDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.profiles, campaignID+"/"+characterID)
	return nil
}

func (m *memDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, s projectionstore.DaggerheartCharacterState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[s.CampaignID+"/"+s.CharacterID] = s
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.states[campaignID+"/"+characterID]
	if !ok {
		return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return s, nil
}

func (m *memDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, s projectionstore.DaggerheartSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[s.CampaignID] = s
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.snapshots[campaignID]
	if !ok {
		return projectionstore.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return s, nil
}

func (m *memDaggerheartStore) PutDaggerheartCountdown(_ context.Context, c projectionstore.DaggerheartCountdown) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.countdowns[c.CampaignID+"/"+c.CountdownID] = c
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.countdowns[campaignID+"/"+countdownID]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return c, nil
}

func (m *memDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []projectionstore.DaggerheartCountdown
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

func (m *memDaggerheartStore) PutDaggerheartAdversary(_ context.Context, a projectionstore.DaggerheartAdversary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.adversaries[a.CampaignID+"/"+a.AdversaryID] = a
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.adversaries[campaignID+"/"+adversaryID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return a, nil
}

func (m *memDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, _ string) ([]projectionstore.DaggerheartAdversary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []projectionstore.DaggerheartAdversary
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

func (m *memDaggerheartStore) PutDaggerheartEnvironmentEntity(_ context.Context, environmentEntity projectionstore.DaggerheartEnvironmentEntity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.environments[environmentEntity.CampaignID+"/"+environmentEntity.EnvironmentEntityID] = environmentEntity
	return nil
}

func (m *memDaggerheartStore) GetDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	environmentEntity, ok := m.environments[campaignID+"/"+environmentEntityID]
	if !ok {
		return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
	}
	return environmentEntity, nil
}

func (m *memDaggerheartStore) ListDaggerheartEnvironmentEntities(_ context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []projectionstore.DaggerheartEnvironmentEntity
	prefix := campaignID + "/"
	for key, value := range m.environments {
		if len(key) <= len(prefix) || key[:len(prefix)] != prefix {
			continue
		}
		if sessionID != "" && value.SessionID != sessionID {
			continue
		}
		if sceneID != "" && value.SceneID != sceneID {
			continue
		}
		result = append(result, value)
	}
	return result, nil
}

func (m *memDaggerheartStore) DeleteDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.environments, campaignID+"/"+environmentEntityID)
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
	clear(m.environments)
}

var _ projectionstore.Store = (*memDaggerheartStore)(nil)

// storeResetter allows ValidateAdapterIdempotency to clear store state
// between event types without depending on system-specific store types.
var _ fmt.Stringer = (*memDaggerheartStore)(nil) // compile guard only

func (m *memDaggerheartStore) String() string { return "memDaggerheartStore" }
