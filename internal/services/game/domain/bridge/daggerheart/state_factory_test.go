package daggerheart

import "testing"

func TestStateFactory_NewSnapshotStateDefaults(t *testing.T) {
	factory := NewStateFactory()
	state, err := factory.NewSnapshotState("camp-1")
	if err != nil {
		t.Fatalf("new snapshot state: %v", err)
	}
	snapshot, ok := state.(SnapshotState)
	if !ok {
		t.Fatalf("expected SnapshotState, got %T", state)
	}
	if snapshot.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want %s", snapshot.CampaignID, "camp-1")
	}
	if snapshot.GMFear != GMFearDefault {
		t.Fatalf("gm fear = %d, want %d", snapshot.GMFear, GMFearDefault)
	}
	if snapshot.CharacterStates == nil {
		t.Fatal("CharacterStates map should be initialized, got nil")
	}
	if snapshot.AdversaryStates == nil {
		t.Fatal("AdversaryStates map should be initialized, got nil")
	}
	if snapshot.CountdownStates == nil {
		t.Fatal("CountdownStates map should be initialized, got nil")
	}
}

func TestSnapshotState_EnsureMaps(t *testing.T) {
	s := SnapshotState{}
	if s.CharacterStates != nil || s.AdversaryStates != nil || s.CountdownStates != nil {
		t.Fatal("expected nil maps before EnsureMaps")
	}
	s.EnsureMaps()
	if s.CharacterStates == nil {
		t.Fatal("CharacterStates should be initialized after EnsureMaps")
	}
	if s.AdversaryStates == nil {
		t.Fatal("AdversaryStates should be initialized after EnsureMaps")
	}
	if s.CountdownStates == nil {
		t.Fatal("CountdownStates should be initialized after EnsureMaps")
	}

	// EnsureMaps should not overwrite existing maps.
	s.CharacterStates["char-1"] = CharacterState{CharacterID: "char-1"}
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
	character, ok := state.(CharacterState)
	if !ok {
		t.Fatalf("expected CharacterState, got %T", state)
	}
	if character.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want %s", character.CampaignID, "camp-1")
	}
	if character.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", character.CharacterID, "char-1")
	}
	if character.Hope != HopeDefault {
		t.Fatalf("hope = %d, want %d", character.Hope, HopeDefault)
	}
	if character.StressMax != StressMaxDefault {
		t.Fatalf("stress max = %d, want %d", character.StressMax, StressMaxDefault)
	}

	state, err = factory.NewCharacterState("camp-1", "npc-1", "npc")
	if err != nil {
		t.Fatalf("new npc state: %v", err)
	}
	character, ok = state.(CharacterState)
	if !ok {
		t.Fatalf("expected CharacterState, got %T", state)
	}
	if character.Hope != 0 {
		t.Fatalf("npc hope = %d, want %d", character.Hope, 0)
	}
	if character.StressMax != 0 {
		t.Fatalf("npc stress max = %d, want %d", character.StressMax, 0)
	}
}
