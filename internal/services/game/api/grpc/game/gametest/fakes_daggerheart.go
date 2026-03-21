package gametest

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// FakeDaggerheartStore is a test double for projectionstore.Store.
type FakeDaggerheartStore struct {
	Profiles            map[string]map[string]projectionstore.DaggerheartCharacterProfile  // campaignID -> characterID -> Profile
	States              map[string]map[string]projectionstore.DaggerheartCharacterState    // campaignID -> characterID -> state
	Snapshots           map[string]projectionstore.DaggerheartSnapshot                     // campaignID -> snapshot
	Countdowns          map[string]map[string]projectionstore.DaggerheartCountdown         // campaignID -> countdownID -> countdown
	Adversaries         map[string]map[string]projectionstore.DaggerheartAdversary         // campaignID -> adversaryID -> adversary
	EnvironmentEntities map[string]map[string]projectionstore.DaggerheartEnvironmentEntity // campaignID -> environmentEntityID -> entity
	StatePuts           map[string]int
	SnapPuts            map[string]int
	PutErr              error
	GetErr              error
}

// NewFakeDaggerheartStore returns a ready-to-use Daggerheart projection fake.
func NewFakeDaggerheartStore() *FakeDaggerheartStore {
	return &FakeDaggerheartStore{
		Profiles:            make(map[string]map[string]projectionstore.DaggerheartCharacterProfile),
		States:              make(map[string]map[string]projectionstore.DaggerheartCharacterState),
		Snapshots:           make(map[string]projectionstore.DaggerheartSnapshot),
		Countdowns:          make(map[string]map[string]projectionstore.DaggerheartCountdown),
		Adversaries:         make(map[string]map[string]projectionstore.DaggerheartAdversary),
		EnvironmentEntities: make(map[string]map[string]projectionstore.DaggerheartEnvironmentEntity),
		StatePuts:           make(map[string]int),
		SnapPuts:            make(map[string]int),
	}
}

func (s *FakeDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, p projectionstore.DaggerheartCharacterProfile) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.Profiles[p.CampaignID] == nil {
		s.Profiles[p.CampaignID] = make(map[string]projectionstore.DaggerheartCharacterProfile)
	}
	s.Profiles[p.CampaignID][p.CharacterID] = p
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCharacterProfile{}, s.GetErr
	}
	byID, ok := s.Profiles[campaignID]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	p, ok := byID[characterID]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *FakeDaggerheartStore) ListDaggerheartCharacterProfiles(_ context.Context, campaignID string, pageSize int, pageToken string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCharacterProfilePage{}, s.GetErr
	}
	if pageSize <= 0 {
		return projectionstore.DaggerheartCharacterProfilePage{}, fmt.Errorf("page size must be greater than zero")
	}
	byID, ok := s.Profiles[campaignID]
	if !ok {
		return projectionstore.DaggerheartCharacterProfilePage{}, nil
	}
	ids := make([]string, 0, len(byID))
	for characterID := range byID {
		if pageToken != "" && characterID <= pageToken {
			continue
		}
		ids = append(ids, characterID)
	}
	sort.Strings(ids)

	page := projectionstore.DaggerheartCharacterProfilePage{
		Profiles: make([]projectionstore.DaggerheartCharacterProfile, 0, min(pageSize, len(ids))),
	}
	for index, characterID := range ids {
		if index >= pageSize {
			page.NextPageToken = ids[pageSize-1]
			break
		}
		page.Profiles = append(page.Profiles, byID[characterID])
	}
	return page, nil
}

func (s *FakeDaggerheartStore) DeleteDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	byID, ok := s.Profiles[campaignID]
	if !ok {
		return nil
	}
	delete(byID, characterID)
	if len(byID) == 0 {
		delete(s.Profiles, campaignID)
	}
	return nil
}

func (s *FakeDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, st projectionstore.DaggerheartCharacterState) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.States[st.CampaignID] == nil {
		s.States[st.CampaignID] = make(map[string]projectionstore.DaggerheartCharacterState)
	}
	s.States[st.CampaignID][st.CharacterID] = st
	s.StatePuts[st.CampaignID]++
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCharacterState{}, s.GetErr
	}
	byID, ok := s.States[campaignID]
	if !ok {
		return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	st, ok := byID[characterID]
	if !ok {
		return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return st, nil
}

func (s *FakeDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap projectionstore.DaggerheartSnapshot) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Snapshots[snap.CampaignID] = snap
	s.SnapPuts[snap.CampaignID]++
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartSnapshot{}, s.GetErr
	}
	snap, ok := s.Snapshots[campaignID]
	if !ok {
		return projectionstore.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (s *FakeDaggerheartStore) PutDaggerheartCountdown(_ context.Context, countdown projectionstore.DaggerheartCountdown) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.Countdowns[countdown.CampaignID] == nil {
		s.Countdowns[countdown.CampaignID] = make(map[string]projectionstore.DaggerheartCountdown)
	}
	s.Countdowns[countdown.CampaignID][countdown.CountdownID] = countdown
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCountdown{}, s.GetErr
	}
	byID, ok := s.Countdowns[campaignID]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	countdown, ok := byID[countdownID]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (s *FakeDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	byID, ok := s.Countdowns[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]projectionstore.DaggerheartCountdown, 0, len(byID))
	for _, countdown := range byID {
		result = append(result, countdown)
	}
	return result, nil
}

func (s *FakeDaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	byID, ok := s.Countdowns[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[countdownID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, countdownID)
	return nil
}

func (s *FakeDaggerheartStore) PutDaggerheartAdversary(_ context.Context, adversary projectionstore.DaggerheartAdversary) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.Adversaries[adversary.CampaignID] == nil {
		s.Adversaries[adversary.CampaignID] = make(map[string]projectionstore.DaggerheartAdversary)
	}
	s.Adversaries[adversary.CampaignID][adversary.AdversaryID] = adversary
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartAdversary{}, s.GetErr
	}
	byID, ok := s.Adversaries[campaignID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	adversary, ok := byID[adversaryID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (s *FakeDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	byID, ok := s.Adversaries[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]projectionstore.DaggerheartAdversary, 0, len(byID))
	for _, adversary := range byID {
		if strings.TrimSpace(sessionID) != "" && adversary.SessionID != sessionID {
			continue
		}
		result = append(result, adversary)
	}
	return result, nil
}

func (s *FakeDaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	byID, ok := s.Adversaries[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[adversaryID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, adversaryID)
	return nil
}

func (s *FakeDaggerheartStore) PutDaggerheartEnvironmentEntity(_ context.Context, entity projectionstore.DaggerheartEnvironmentEntity) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.EnvironmentEntities[entity.CampaignID] == nil {
		s.EnvironmentEntities[entity.CampaignID] = make(map[string]projectionstore.DaggerheartEnvironmentEntity)
	}
	s.EnvironmentEntities[entity.CampaignID][entity.EnvironmentEntityID] = entity
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, s.GetErr
	}
	byID, ok := s.EnvironmentEntities[campaignID]
	if !ok {
		return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
	}
	entity, ok := byID[environmentEntityID]
	if !ok {
		return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
	}
	return entity, nil
}

func (s *FakeDaggerheartStore) ListDaggerheartEnvironmentEntities(_ context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	byID, ok := s.EnvironmentEntities[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]projectionstore.DaggerheartEnvironmentEntity, 0, len(byID))
	for _, entity := range byID {
		if strings.TrimSpace(sessionID) != "" && entity.SessionID != sessionID {
			continue
		}
		if strings.TrimSpace(sceneID) != "" && entity.SceneID != sceneID {
			continue
		}
		result = append(result, entity)
	}
	return result, nil
}

func (s *FakeDaggerheartStore) DeleteDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	byID, ok := s.EnvironmentEntities[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[environmentEntityID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, environmentEntityID)
	return nil
}
