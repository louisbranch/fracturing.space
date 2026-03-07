package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
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
	profiles    map[string]storage.DaggerheartCharacterProfile
	states      map[string]storage.DaggerheartCharacterState
	snapshots   map[string]storage.DaggerheartSnapshot
	countdowns  map[string]storage.DaggerheartCountdown
	adversaries map[string]storage.DaggerheartAdversary
}

func newProjectionDaggerheartStore() *projectionDaggerheartStore {
	return &projectionDaggerheartStore{
		profiles:    make(map[string]storage.DaggerheartCharacterProfile),
		states:      make(map[string]storage.DaggerheartCharacterState),
		snapshots:   make(map[string]storage.DaggerheartSnapshot),
		countdowns:  make(map[string]storage.DaggerheartCountdown),
		adversaries: make(map[string]storage.DaggerheartAdversary),
	}
}

func newProjectionApplier(campaignStore *projectionCampaignStore, daggerheartStore *projectionDaggerheartStore) Applier {
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(daggerheart.NewAdapter(daggerheartStore)); err != nil {
		panic(err)
	}
	return Applier{Campaign: campaignStore, Adapters: registry}
}

func (s *projectionDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, profile storage.DaggerheartCharacterProfile) error {
	key := profile.CampaignID + ":" + profile.CharacterID
	s.profiles[key] = profile
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	key := campaignID + ":" + characterID
	profile, ok := s.profiles[key]
	if !ok {
		return storage.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return profile, nil
}

func (s *projectionDaggerheartStore) DeleteDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) error {
	key := campaignID + ":" + characterID
	delete(s.profiles, key)
	return nil
}

func (s *projectionDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, state storage.DaggerheartCharacterState) error {
	key := state.CampaignID + ":" + state.CharacterID
	s.states[key] = state
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	key := campaignID + ":" + characterID
	state, ok := s.states[key]
	if !ok {
		return storage.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return state, nil
}

func (s *projectionDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap storage.DaggerheartSnapshot) error {
	s.snapshots[snap.CampaignID] = snap
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	snap, ok := s.snapshots[campaignID]
	if !ok {
		return storage.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (s *projectionDaggerheartStore) PutDaggerheartCountdown(_ context.Context, countdown storage.DaggerheartCountdown) error {
	key := countdown.CampaignID + ":" + countdown.CountdownID
	s.countdowns[key] = countdown
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	key := campaignID + ":" + countdownID
	countdown, ok := s.countdowns[key]
	if !ok {
		return storage.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (s *projectionDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]storage.DaggerheartCountdown, error) {
	results := make([]storage.DaggerheartCountdown, 0)
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

func (s *projectionDaggerheartStore) PutDaggerheartAdversary(_ context.Context, adversary storage.DaggerheartAdversary) error {
	key := adversary.CampaignID + ":" + adversary.AdversaryID
	s.adversaries[key] = adversary
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	key := campaignID + ":" + adversaryID
	adversary, ok := s.adversaries[key]
	if !ok {
		return storage.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (s *projectionDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]storage.DaggerheartAdversary, error) {
	results := make([]storage.DaggerheartAdversary, 0)
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
