package daggerheart

import (
	"context"
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestSessionStartReadinessStateLoader_LoadsSnapshotAndProfiles(t *testing.T) {
	store := &readinessStoreStub{
		snapshot: projectionstore.DaggerheartSnapshot{
			CampaignID: "camp-1",
			GMFear:     4,
		},
		profiles: []projectionstore.DaggerheartCharacterProfile{{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Level:       2,
		}},
	}

	state, err := sessionStartReadinessStateLoader{}.LoadSessionStartReadinessState(
		context.Background(),
		"camp-1",
		testProjectionStores{daggerheart: store},
		aggregate.State{
			Characters: map[ids.CharacterID]character.State{
				"char-1": {CharacterID: "char-1"},
			},
			Systems: map[module.Key]any{},
		},
	)
	if err != nil {
		t.Fatalf("LoadSessionStartReadinessState() error = %v", err)
	}

	systemState, ok := state.Systems[module.Key{ID: SystemID, Version: SystemVersion}]
	if !ok {
		t.Fatal("daggerheart system state not found")
	}
	snapshot, ok := systemState.(daggerheartstate.SnapshotState)
	if !ok {
		t.Fatalf("system state type = %T, want SnapshotState", systemState)
	}
	if snapshot.GMFear != 4 {
		t.Fatalf("snapshot gm fear = %d, want 4", snapshot.GMFear)
	}
	if got := snapshot.CharacterProfiles["char-1"].Level; got != 2 {
		t.Fatalf("profile level = %d, want 2", got)
	}
}

func TestSessionStartReadinessStateLoader_RequiresStore(t *testing.T) {
	_, err := sessionStartReadinessStateLoader{}.LoadSessionStartReadinessState(
		context.Background(),
		"camp-1",
		testProjectionStores{},
		aggregate.State{},
	)
	if err == nil || err.Error() != "daggerheart projection store is not configured" {
		t.Fatalf("LoadSessionStartReadinessState() error = %v, want missing store error", err)
	}
}

func TestSessionStartReadinessStateLoader_WrapsSnapshotReadFailure(t *testing.T) {
	store := &readinessStoreStub{getErr: errors.New("boom")}

	_, err := sessionStartReadinessStateLoader{}.LoadSessionStartReadinessState(
		context.Background(),
		"camp-1",
		testProjectionStores{daggerheart: store},
		aggregate.State{
			Characters: map[ids.CharacterID]character.State{
				"char-1": {CharacterID: "char-1"},
			},
			Systems: map[module.Key]any{},
		},
	)
	if err == nil || err.Error() != "get daggerheart snapshot: boom" {
		t.Fatalf("LoadSessionStartReadinessState() error = %v, want wrapped snapshot read error", err)
	}
}

type testProjectionStores struct {
	daggerheart projectionstore.Store
}

func (s testProjectionStores) DaggerheartProjectionStore() projectionstore.Store {
	return s.daggerheart
}

type readinessStoreStub struct {
	snapshot projectionstore.DaggerheartSnapshot
	profiles []projectionstore.DaggerheartCharacterProfile
	getErr   error
}

func (s *readinessStoreStub) PutDaggerheartCharacterProfile(context.Context, projectionstore.DaggerheartCharacterProfile) error {
	return nil
}

func (s *readinessStoreStub) GetDaggerheartCharacterProfile(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
	return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
}

func (s *readinessStoreStub) ListDaggerheartCharacterProfiles(context.Context, string, int, string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	if s.getErr != nil {
		return projectionstore.DaggerheartCharacterProfilePage{}, s.getErr
	}
	return projectionstore.DaggerheartCharacterProfilePage{Profiles: s.profiles}, nil
}

func (s *readinessStoreStub) DeleteDaggerheartCharacterProfile(context.Context, string, string) error {
	return nil
}

func (s *readinessStoreStub) PutDaggerheartCharacterState(context.Context, projectionstore.DaggerheartCharacterState) error {
	return nil
}

func (s *readinessStoreStub) GetDaggerheartCharacterState(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
	return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
}

func (s *readinessStoreStub) PutDaggerheartSnapshot(context.Context, projectionstore.DaggerheartSnapshot) error {
	return nil
}

func (s *readinessStoreStub) GetDaggerheartSnapshot(context.Context, string) (projectionstore.DaggerheartSnapshot, error) {
	if s.getErr != nil {
		return projectionstore.DaggerheartSnapshot{}, s.getErr
	}
	if s.snapshot.CampaignID == "" {
		return projectionstore.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return s.snapshot, nil
}

func (s *readinessStoreStub) PutDaggerheartCountdown(context.Context, projectionstore.DaggerheartCountdown) error {
	return nil
}

func (s *readinessStoreStub) GetDaggerheartCountdown(context.Context, string, string) (projectionstore.DaggerheartCountdown, error) {
	return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
}

func (s *readinessStoreStub) ListDaggerheartCountdowns(context.Context, string) ([]projectionstore.DaggerheartCountdown, error) {
	return nil, nil
}

func (s *readinessStoreStub) DeleteDaggerheartCountdown(context.Context, string, string) error {
	return nil
}

func (s *readinessStoreStub) PutDaggerheartAdversary(context.Context, projectionstore.DaggerheartAdversary) error {
	return nil
}

func (s *readinessStoreStub) GetDaggerheartAdversary(context.Context, string, string) (projectionstore.DaggerheartAdversary, error) {
	return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
}

func (s *readinessStoreStub) ListDaggerheartAdversaries(context.Context, string, string) ([]projectionstore.DaggerheartAdversary, error) {
	return nil, nil
}

func (s *readinessStoreStub) DeleteDaggerheartAdversary(context.Context, string, string) error {
	return nil
}

func (s *readinessStoreStub) PutDaggerheartEnvironmentEntity(context.Context, projectionstore.DaggerheartEnvironmentEntity) error {
	return nil
}

func (s *readinessStoreStub) GetDaggerheartEnvironmentEntity(context.Context, string, string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
}

func (s *readinessStoreStub) ListDaggerheartEnvironmentEntities(context.Context, string, string, string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	return nil, nil
}

func (s *readinessStoreStub) DeleteDaggerheartEnvironmentEntity(context.Context, string, string) error {
	return nil
}

var _ projectionstore.Store = (*readinessStoreStub)(nil)
