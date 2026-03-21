package adapter

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type profileStoreStub struct {
	profiles            map[string]projectionstore.DaggerheartCharacterProfile
	states              map[string]projectionstore.DaggerheartCharacterState
	snapshot            projectionstore.DaggerheartSnapshot
	countdowns          map[string]projectionstore.DaggerheartCountdown
	adversaries         map[string]projectionstore.DaggerheartAdversary
	environmentEntities map[string]projectionstore.DaggerheartEnvironmentEntity

	putCharacterProfileErr     error
	getCharacterProfileErr     error
	deleteCharacterProfileErr  error
	putCharacterStateErr       error
	getCharacterStateErr       error
	putSnapshotErr             error
	getSnapshotErr             error
	putCountdownErr            error
	getCountdownErr            error
	deleteCountdownErr         error
	putAdversaryErr            error
	getAdversaryErr            error
	deleteAdversaryErr         error
	putEnvironmentEntityErr    error
	getEnvironmentEntityErr    error
	deleteEnvironmentEntityErr error
}

func newProfileStoreStub() *profileStoreStub {
	return &profileStoreStub{
		profiles:            map[string]projectionstore.DaggerheartCharacterProfile{},
		states:              map[string]projectionstore.DaggerheartCharacterState{},
		countdowns:          map[string]projectionstore.DaggerheartCountdown{},
		adversaries:         map[string]projectionstore.DaggerheartAdversary{},
		environmentEntities: map[string]projectionstore.DaggerheartEnvironmentEntity{},
	}
}

func profileKey(campaignID, entityID string) string { return campaignID + "/" + entityID }

func (s *profileStoreStub) PutDaggerheartCharacterProfile(_ context.Context, profile projectionstore.DaggerheartCharacterProfile) error {
	if s.putCharacterProfileErr != nil {
		return s.putCharacterProfileErr
	}
	s.profiles[profileKey(profile.CampaignID, profile.CharacterID)] = profile
	return nil
}

func (s *profileStoreStub) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	if s.getCharacterProfileErr != nil {
		return projectionstore.DaggerheartCharacterProfile{}, s.getCharacterProfileErr
	}
	profile, ok := s.profiles[profileKey(campaignID, characterID)]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return profile, nil
}

func (s *profileStoreStub) ListDaggerheartCharacterProfiles(_ context.Context, _ string, _ int, _ string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	return projectionstore.DaggerheartCharacterProfilePage{}, nil
}

func (s *profileStoreStub) DeleteDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) error {
	if s.deleteCharacterProfileErr != nil {
		return s.deleteCharacterProfileErr
	}
	delete(s.profiles, profileKey(campaignID, characterID))
	return nil
}

func (s *profileStoreStub) PutDaggerheartCharacterState(_ context.Context, state projectionstore.DaggerheartCharacterState) error {
	if s.putCharacterStateErr != nil {
		return s.putCharacterStateErr
	}
	s.states[profileKey(state.CampaignID, state.CharacterID)] = state
	return nil
}

func (s *profileStoreStub) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if s.getCharacterStateErr != nil {
		return projectionstore.DaggerheartCharacterState{}, s.getCharacterStateErr
	}
	state, ok := s.states[profileKey(campaignID, characterID)]
	if !ok {
		return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return state, nil
}

func (s *profileStoreStub) PutDaggerheartSnapshot(_ context.Context, snap projectionstore.DaggerheartSnapshot) error {
	if s.putSnapshotErr != nil {
		return s.putSnapshotErr
	}
	s.snapshot = snap
	return nil
}

func (s *profileStoreStub) GetDaggerheartSnapshot(_ context.Context, _ string) (projectionstore.DaggerheartSnapshot, error) {
	if s.getSnapshotErr != nil {
		return projectionstore.DaggerheartSnapshot{}, s.getSnapshotErr
	}
	if s.snapshot == (projectionstore.DaggerheartSnapshot{}) {
		return projectionstore.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return s.snapshot, nil
}

func (s *profileStoreStub) PutDaggerheartCountdown(_ context.Context, countdown projectionstore.DaggerheartCountdown) error {
	if s.putCountdownErr != nil {
		return s.putCountdownErr
	}
	s.countdowns[profileKey(countdown.CampaignID, countdown.CountdownID)] = countdown
	return nil
}

func (s *profileStoreStub) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	if s.getCountdownErr != nil {
		return projectionstore.DaggerheartCountdown{}, s.getCountdownErr
	}
	countdown, ok := s.countdowns[profileKey(campaignID, countdownID)]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (s *profileStoreStub) ListDaggerheartCountdowns(_ context.Context, _ string) ([]projectionstore.DaggerheartCountdown, error) {
	return nil, nil
}

func (s *profileStoreStub) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	if s.deleteCountdownErr != nil {
		return s.deleteCountdownErr
	}
	delete(s.countdowns, profileKey(campaignID, countdownID))
	return nil
}

func (s *profileStoreStub) PutDaggerheartAdversary(_ context.Context, adversary projectionstore.DaggerheartAdversary) error {
	if s.putAdversaryErr != nil {
		return s.putAdversaryErr
	}
	s.adversaries[profileKey(adversary.CampaignID, adversary.AdversaryID)] = adversary
	return nil
}

func (s *profileStoreStub) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if s.getAdversaryErr != nil {
		return projectionstore.DaggerheartAdversary{}, s.getAdversaryErr
	}
	adversary, ok := s.adversaries[profileKey(campaignID, adversaryID)]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (s *profileStoreStub) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
	adversaries := make([]projectionstore.DaggerheartAdversary, 0, len(s.adversaries))
	for _, adversary := range s.adversaries {
		if adversary.CampaignID != campaignID {
			continue
		}
		if sessionID != "" && adversary.SessionID != sessionID {
			continue
		}
		adversaries = append(adversaries, adversary)
	}
	return adversaries, nil
}

func (s *profileStoreStub) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	if s.deleteAdversaryErr != nil {
		return s.deleteAdversaryErr
	}
	delete(s.adversaries, profileKey(campaignID, adversaryID))
	return nil
}

func (s *profileStoreStub) PutDaggerheartEnvironmentEntity(_ context.Context, environmentEntity projectionstore.DaggerheartEnvironmentEntity) error {
	if s.putEnvironmentEntityErr != nil {
		return s.putEnvironmentEntityErr
	}
	s.environmentEntities[profileKey(environmentEntity.CampaignID, environmentEntity.EnvironmentEntityID)] = environmentEntity
	return nil
}

func (s *profileStoreStub) GetDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	if s.getEnvironmentEntityErr != nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, s.getEnvironmentEntityErr
	}
	environmentEntity, ok := s.environmentEntities[profileKey(campaignID, environmentEntityID)]
	if !ok {
		return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
	}
	return environmentEntity, nil
}

func (s *profileStoreStub) ListDaggerheartEnvironmentEntities(_ context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	entities := make([]projectionstore.DaggerheartEnvironmentEntity, 0, len(s.environmentEntities))
	for _, entity := range s.environmentEntities {
		if entity.CampaignID != campaignID {
			continue
		}
		if sessionID != "" && entity.SessionID != sessionID {
			continue
		}
		if sceneID != "" && entity.SceneID != sceneID {
			continue
		}
		entities = append(entities, entity)
	}
	return entities, nil
}

func (s *profileStoreStub) DeleteDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) error {
	if s.deleteEnvironmentEntityErr != nil {
		return s.deleteEnvironmentEntityErr
	}
	delete(s.environmentEntities, profileKey(campaignID, environmentEntityID))
	return nil
}
