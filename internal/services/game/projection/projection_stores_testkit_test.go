package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type projectionCampaignStore struct {
	campaigns map[string]storage.CampaignRecord
}

func newProjectionCampaignStore() *projectionCampaignStore {
	return &projectionCampaignStore{campaigns: make(map[string]storage.CampaignRecord)}
}

func (s *projectionCampaignStore) Put(_ context.Context, c storage.CampaignRecord) error {
	s.campaigns[c.ID] = c
	return nil
}

func (s *projectionCampaignStore) Get(_ context.Context, id string) (storage.CampaignRecord, error) {
	c, ok := s.campaigns[id]
	if !ok {
		return storage.CampaignRecord{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *projectionCampaignStore) List(context.Context, int, string) (storage.CampaignPage, error) {
	return storage.CampaignPage{}, nil
}

type projectionParticipantStore struct {
	participants map[string]storage.ParticipantRecord
}

func newProjectionParticipantStore() *projectionParticipantStore {
	return &projectionParticipantStore{participants: make(map[string]storage.ParticipantRecord)}
}

func (s *projectionParticipantStore) PutParticipant(_ context.Context, p storage.ParticipantRecord) error {
	s.participants[p.CampaignID+":"+p.ID] = p
	return nil
}

func (s *projectionParticipantStore) GetParticipant(_ context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
	key := campaignID + ":" + participantID
	p, ok := s.participants[key]
	if !ok {
		return storage.ParticipantRecord{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *projectionParticipantStore) DeleteParticipant(_ context.Context, campaignID, participantID string) error {
	key := campaignID + ":" + participantID
	if _, ok := s.participants[key]; !ok {
		return fmt.Errorf("not found")
	}
	delete(s.participants, key)
	return nil
}

func (s *projectionParticipantStore) CountParticipants(_ context.Context, campaignID string) (int, error) {
	count := 0
	for key := range s.participants {
		if strings.HasPrefix(key, campaignID+":") {
			count++
		}
	}
	return count, nil
}

func (s *projectionParticipantStore) ListParticipantsByCampaign(context.Context, string) ([]storage.ParticipantRecord, error) {
	return nil, nil
}

func (s *projectionParticipantStore) ListCampaignIDsByUser(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *projectionParticipantStore) ListCampaignIDsByParticipant(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *projectionParticipantStore) ListParticipants(context.Context, string, int, string) (storage.ParticipantPage, error) {
	return storage.ParticipantPage{}, nil
}

type projectionDaggerheartStore struct {
	profiles     map[string]projectionstore.DaggerheartCharacterProfile
	states       map[string]projectionstore.DaggerheartCharacterState
	snapshots    map[string]projectionstore.DaggerheartSnapshot
	countdowns   map[string]projectionstore.DaggerheartCountdown
	adversaries  map[string]projectionstore.DaggerheartAdversary
	environments map[string]projectionstore.DaggerheartEnvironmentEntity
}

func newProjectionDaggerheartStore() *projectionDaggerheartStore {
	return &projectionDaggerheartStore{
		profiles:     make(map[string]projectionstore.DaggerheartCharacterProfile),
		states:       make(map[string]projectionstore.DaggerheartCharacterState),
		snapshots:    make(map[string]projectionstore.DaggerheartSnapshot),
		countdowns:   make(map[string]projectionstore.DaggerheartCountdown),
		adversaries:  make(map[string]projectionstore.DaggerheartAdversary),
		environments: make(map[string]projectionstore.DaggerheartEnvironmentEntity),
	}
}

func newProjectionApplier(campaignStore *projectionCampaignStore, daggerheartStore *projectionDaggerheartStore) Applier {
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(daggerheart.NewAdapter(daggerheartStore)); err != nil {
		panic(err)
	}
	return Applier{Campaign: campaignStore, Adapters: registry}
}

func (s *projectionDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, profile projectionstore.DaggerheartCharacterProfile) error {
	key := profile.CampaignID + ":" + profile.CharacterID
	s.profiles[key] = profile
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	key := campaignID + ":" + characterID
	profile, ok := s.profiles[key]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return profile, nil
}

func (s *projectionDaggerheartStore) ListDaggerheartCharacterProfiles(_ context.Context, campaignID string, _ int, _ string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	page := projectionstore.DaggerheartCharacterProfilePage{
		Profiles: make([]projectionstore.DaggerheartCharacterProfile, 0),
	}
	prefix := campaignID + ":"
	for key, profile := range s.profiles {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			page.Profiles = append(page.Profiles, profile)
		}
	}
	return page, nil
}

func (s *projectionDaggerheartStore) DeleteDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) error {
	key := campaignID + ":" + characterID
	delete(s.profiles, key)
	return nil
}

func (s *projectionDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, state projectionstore.DaggerheartCharacterState) error {
	key := state.CampaignID + ":" + state.CharacterID
	s.states[key] = state
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	key := campaignID + ":" + characterID
	state, ok := s.states[key]
	if !ok {
		return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return state, nil
}

func (s *projectionDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap projectionstore.DaggerheartSnapshot) error {
	s.snapshots[snap.CampaignID] = snap
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	snap, ok := s.snapshots[campaignID]
	if !ok {
		return projectionstore.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (s *projectionDaggerheartStore) PutDaggerheartCountdown(_ context.Context, countdown projectionstore.DaggerheartCountdown) error {
	key := countdown.CampaignID + ":" + countdown.CountdownID
	s.countdowns[key] = countdown
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	key := campaignID + ":" + countdownID
	countdown, ok := s.countdowns[key]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (s *projectionDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
	results := make([]projectionstore.DaggerheartCountdown, 0)
	for _, countdown := range s.countdowns {
		if countdown.CampaignID == campaignID {
			results = append(results, countdown)
		}
	}
	return results, nil
}

func (s *projectionDaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	key := campaignID + ":" + countdownID
	if _, ok := s.countdowns[key]; !ok {
		return storage.ErrNotFound
	}
	delete(s.countdowns, key)
	return nil
}

func (s *projectionDaggerheartStore) PutDaggerheartAdversary(_ context.Context, adversary projectionstore.DaggerheartAdversary) error {
	key := adversary.CampaignID + ":" + adversary.AdversaryID
	s.adversaries[key] = adversary
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	key := campaignID + ":" + adversaryID
	adversary, ok := s.adversaries[key]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (s *projectionDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
	results := make([]projectionstore.DaggerheartAdversary, 0)
	for _, adversary := range s.adversaries {
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

func (s *projectionDaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	key := campaignID + ":" + adversaryID
	if _, ok := s.adversaries[key]; !ok {
		return storage.ErrNotFound
	}
	delete(s.adversaries, key)
	return nil
}

func (s *projectionDaggerheartStore) PutDaggerheartEnvironmentEntity(_ context.Context, environmentEntity projectionstore.DaggerheartEnvironmentEntity) error {
	s.environments[environmentEntity.CampaignID+":"+environmentEntity.EnvironmentEntityID] = environmentEntity
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	environmentEntity, ok := s.environments[campaignID+":"+environmentEntityID]
	if !ok {
		return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
	}
	return environmentEntity, nil
}

func (s *projectionDaggerheartStore) ListDaggerheartEnvironmentEntities(_ context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	results := make([]projectionstore.DaggerheartEnvironmentEntity, 0)
	for _, environmentEntity := range s.environments {
		if environmentEntity.CampaignID != campaignID {
			continue
		}
		if strings.TrimSpace(sessionID) != "" && environmentEntity.SessionID != sessionID {
			continue
		}
		if strings.TrimSpace(sceneID) != "" && environmentEntity.SceneID != sceneID {
			continue
		}
		results = append(results, environmentEntity)
	}
	return results, nil
}

func (s *projectionDaggerheartStore) DeleteDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) error {
	key := campaignID + ":" + environmentEntityID
	if _, ok := s.environments[key]; !ok {
		return storage.ErrNotFound
	}
	delete(s.environments, key)
	return nil
}
