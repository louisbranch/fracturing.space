package adapter

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type profileStoreStub struct {
	profiles   map[string]projectionstore.DaggerheartCharacterProfile
	states     map[string]projectionstore.DaggerheartCharacterState
	snapshot   projectionstore.DaggerheartSnapshot
	countdowns map[string]projectionstore.DaggerheartCountdown

	putCharacterProfileErr    error
	getCharacterProfileErr    error
	deleteCharacterProfileErr error
	putCharacterStateErr      error
	getCharacterStateErr      error
	putSnapshotErr            error
	getSnapshotErr            error
	putCountdownErr           error
	getCountdownErr           error
	deleteCountdownErr        error
}

func newProfileStoreStub() *profileStoreStub {
	return &profileStoreStub{
		profiles:   map[string]projectionstore.DaggerheartCharacterProfile{},
		states:     map[string]projectionstore.DaggerheartCharacterState{},
		countdowns: map[string]projectionstore.DaggerheartCountdown{},
	}
}

func profileKey(campaignID, characterID string) string { return campaignID + "/" + characterID }

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

func (s *profileStoreStub) PutDaggerheartAdversary(_ context.Context, _ projectionstore.DaggerheartAdversary) error {
	return nil
}

func (s *profileStoreStub) GetDaggerheartAdversary(_ context.Context, _, _ string) (projectionstore.DaggerheartAdversary, error) {
	return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
}

func (s *profileStoreStub) ListDaggerheartAdversaries(_ context.Context, _, _ string) ([]projectionstore.DaggerheartAdversary, error) {
	return nil, nil
}

func (s *profileStoreStub) DeleteDaggerheartAdversary(_ context.Context, _, _ string) error {
	return nil
}

func (s *profileStoreStub) PutDaggerheartEnvironmentEntity(_ context.Context, _ projectionstore.DaggerheartEnvironmentEntity) error {
	return nil
}

func (s *profileStoreStub) GetDaggerheartEnvironmentEntity(_ context.Context, _, _ string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
}

func (s *profileStoreStub) ListDaggerheartEnvironmentEntities(_ context.Context, _, _, _ string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	return nil, nil
}

func (s *profileStoreStub) DeleteDaggerheartEnvironmentEntity(_ context.Context, _, _ string) error {
	return nil
}

func TestPutCharacterProfile_SeedsArmorStateFromNormalizedProfile(t *testing.T) {
	t.Parallel()

	store := newProfileStoreStub()
	a := NewAdapter(store, nil)

	err := a.PutCharacterProfile(context.Background(), "camp-1", "char-1", daggerheartstate.CharacterProfile{
		Level:           1,
		HpMax:           7,
		StressMax:       6,
		Evasion:         9,
		MajorThreshold:  8,
		SevereThreshold: 12,
		Proficiency:     1,
		ArmorScore:      4,
		StartingArmorID: "armor.chainmail-armor",
	})
	if err != nil {
		t.Fatalf("PutCharacterProfile returned error: %v", err)
	}

	profile, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("GetDaggerheartCharacterProfile returned error: %v", err)
	}
	if profile.EquippedArmorID != "armor.chainmail-armor" {
		t.Fatalf("equipped armor id = %q, want %q", profile.EquippedArmorID, "armor.chainmail-armor")
	}
	if profile.ArmorMax != 4 {
		t.Fatalf("armor max = %d, want 4", profile.ArmorMax)
	}

	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("GetDaggerheartCharacterState returned error: %v", err)
	}
	if state.Armor != 4 {
		t.Fatalf("state armor = %d, want 4", state.Armor)
	}
}
