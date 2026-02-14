package daggerheart

import "testing"

func TestNewLoadout(t *testing.T) {
	_, err := NewLoadout([]string{"a", "b", "c", "d", "e", "f"}, nil)
	if err != ErrLoadoutFull {
		t.Fatalf("expected ErrLoadoutFull, got %v", err)
	}

	_, err = NewLoadout([]string{"a"}, []string{"a"})
	if err != ErrDuplicateCard {
		t.Fatalf("expected ErrDuplicateCard, got %v", err)
	}
}

func TestMoveToActive(t *testing.T) {
	loadout, err := NewLoadout([]string{"a"}, []string{"b"})
	if err != nil {
		t.Fatalf("NewLoadout returned error: %v", err)
	}
	updated, err := loadout.MoveToActive("b")
	if err != nil {
		t.Fatalf("MoveToActive returned error: %v", err)
	}
	if len(updated.Active) != 2 || len(updated.Vault) != 0 {
		t.Fatalf("unexpected loadout state: %+v", updated)
	}
}

func TestMoveToVault(t *testing.T) {
	loadout, err := NewLoadout([]string{"a", "b"}, []string{})
	if err != nil {
		t.Fatalf("NewLoadout returned error: %v", err)
	}
	updated, err := loadout.MoveToVault("a")
	if err != nil {
		t.Fatalf("MoveToVault returned error: %v", err)
	}
	if len(updated.Active) != 1 || len(updated.Vault) != 1 {
		t.Fatalf("unexpected loadout state: %+v", updated)
	}
}

func TestMoveToActiveWithRecall(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		Hope:        2,
		HopeMax:     HopeMax,
		Stress:      2,
		StressMax:   6,
		Armor:       0,
		ArmorMax:    0,
	})
	loadout, err := NewLoadout([]string{}, []string{"card-1"})
	if err != nil {
		t.Fatalf("NewLoadout returned error: %v", err)
	}
	updated, err := loadout.MoveToActiveWithRecall(DomainCard{ID: "card-1", RecallCost: 1}, state, false)
	if err != nil {
		t.Fatalf("MoveToActiveWithRecall returned error: %v", err)
	}
	if state.Stress() != 1 {
		t.Fatalf("stress = %d, want 1", state.Stress())
	}
	if len(updated.Active) != 1 {
		t.Fatalf("expected card in active loadout")
	}
}
