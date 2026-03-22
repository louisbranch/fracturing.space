package daggerheart

import (
	"testing"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestStateFactory_NewSnapshotStateDefaults(t *testing.T) {
	factory := NewStateFactory()
	state, err := factory.NewSnapshotState("camp-1")
	if err != nil {
		t.Fatalf("new snapshot state: %v", err)
	}
	snapshot, ok := state.(daggerheartstate.SnapshotState)
	if !ok {
		t.Fatalf("expected daggerheartstate.SnapshotState, got %T", state)
	}
	if snapshot.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want %s", snapshot.CampaignID, "camp-1")
	}
	if snapshot.GMFear != daggerheartstate.GMFearDefault {
		t.Fatalf("gm fear = %d, want %d", snapshot.GMFear, daggerheartstate.GMFearDefault)
	}
	if snapshot.CharacterStates == nil {
		t.Fatal("CharacterStates map should be initialized, got nil")
	}
	if snapshot.AdversaryStates == nil {
		t.Fatal("AdversaryStates map should be initialized, got nil")
	}
	if snapshot.SceneCountdownStates == nil || snapshot.CampaignCountdownStates == nil {
		t.Fatal("countdown maps should be initialized, got nil")
	}
}

func TestSnapshotState_EnsureMaps(t *testing.T) {
	s := daggerheartstate.SnapshotState{}
	if s.CharacterStates != nil || s.AdversaryStates != nil || s.SceneCountdownStates != nil || s.CampaignCountdownStates != nil {
		t.Fatal("expected nil maps before EnsureMaps")
	}
	s.EnsureMaps()
	if s.CharacterStates == nil {
		t.Fatal("CharacterStates should be initialized after EnsureMaps")
	}
	if s.AdversaryStates == nil {
		t.Fatal("AdversaryStates should be initialized after EnsureMaps")
	}
	if s.SceneCountdownStates == nil || s.CampaignCountdownStates == nil {
		t.Fatal("countdown maps should be initialized after EnsureMaps")
	}

	// EnsureMaps should not overwrite existing maps.
	s.CharacterStates["char-1"] = daggerheartstate.CharacterState{CharacterID: "char-1"}
	s.EnsureMaps()
	if _, ok := s.CharacterStates["char-1"]; !ok {
		t.Fatal("EnsureMaps overwrote existing CharacterStates map")
	}
}

func TestStateFactory_NewCharacterStateDefaults(t *testing.T) {
	factory := NewStateFactory()
	state, err := factory.NewCharacterState("camp-1", "char-1", "pc")
	if err != nil {
		t.Fatalf("new character state: %v", err)
	}
	character, ok := state.(daggerheartstate.CharacterState)
	if !ok {
		t.Fatalf("expected daggerheartstate.CharacterState, got %T", state)
	}
	if character.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want %s", character.CampaignID, "camp-1")
	}
	if character.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", character.CharacterID, "char-1")
	}
	if character.Hope != daggerheartstate.HopeDefault {
		t.Fatalf("hope = %d, want %d", character.Hope, daggerheartstate.HopeDefault)
	}
	if character.StressMax != daggerheartstate.StressMaxDefault {
		t.Fatalf("stress max = %d, want %d", character.StressMax, daggerheartstate.StressMaxDefault)
	}

	state, err = factory.NewCharacterState("camp-1", "npc-1", "npc")
	if err != nil {
		t.Fatalf("new npc state: %v", err)
	}
	character, ok = state.(daggerheartstate.CharacterState)
	if !ok {
		t.Fatalf("expected daggerheartstate.CharacterState, got %T", state)
	}
	if character.Hope != 0 {
		t.Fatalf("npc hope = %d, want %d", character.Hope, 0)
	}
	if character.StressMax != 0 {
		t.Fatalf("npc stress max = %d, want %d", character.StressMax, 0)
	}
}
