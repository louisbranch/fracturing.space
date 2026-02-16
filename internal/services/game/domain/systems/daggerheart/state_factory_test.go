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
