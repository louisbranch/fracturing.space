package daggerheart

import (
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestRegistrySystemExposesSessionStartReadinessLoader(t *testing.T) {
	system := NewRegistrySystem()
	if system.SessionStartReadinessStateLoader() == nil {
		t.Fatal("expected session-start readiness state loader")
	}
}

func TestBindSessionStartBootstrapRequiresFactoryWhenSnapshotMissing(t *testing.T) {
	_, err := bindSessionStartBootstrap(nil, "camp-1", map[module.Key]any{})
	if err == nil || err.Error() != "daggerheart state factory is not configured" {
		t.Fatalf("bindSessionStartBootstrap() error = %v, want missing factory error", err)
	}
}

func TestBindSessionStartBootstrapSeedsSnapshotFromFactory(t *testing.T) {
	emitter, err := bindSessionStartBootstrap(snapshotFactoryStub{
		snapshot: daggerheartstate.SnapshotState{CampaignID: "camp-1"},
	}, "camp-1", map[module.Key]any{})
	if err != nil {
		t.Fatalf("bindSessionStartBootstrap() error = %v", err)
	}
	if emitter == nil {
		t.Fatal("expected session start bootstrap emitter")
	}
}

func TestBindSessionStartBootstrapWrapsFactoryError(t *testing.T) {
	_, err := bindSessionStartBootstrap(snapshotFactoryStub{err: errors.New("boom")}, "camp-1", map[module.Key]any{})
	if err == nil || err.Error() != "daggerheart state factory NewSnapshotState: boom" {
		t.Fatalf("bindSessionStartBootstrap() error = %v, want wrapped factory error", err)
	}
}

type snapshotFactoryStub struct {
	snapshot any
	err      error
}

func (s snapshotFactoryStub) NewCharacterState(ids.CampaignID, ids.CharacterID, string) (any, error) {
	return nil, errors.New("unexpected NewCharacterState call")
}

func (s snapshotFactoryStub) NewSnapshotState(ids.CampaignID) (any, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.snapshot, nil
}
