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

func TestMoveToActiveWithRecallAtRest(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		HP: 6, HPMax: 6, Stress: 2, StressMax: 6, HopeMax: HopeMax,
	})
	loadout, _ := NewLoadout([]string{}, []string{"card-1"})
	_, err := loadout.MoveToActiveWithRecall(DomainCard{ID: "card-1", RecallCost: 3}, state, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Stress() != 2 {
		t.Fatalf("stress = %d, want 2 (no cost at rest)", state.Stress())
	}
}

func TestMoveToActiveCardNotFound(t *testing.T) {
	loadout, _ := NewLoadout([]string{"a"}, []string{})
	_, err := loadout.MoveToActive("missing")
	if err != ErrCardNotFound {
		t.Fatalf("expected ErrCardNotFound, got %v", err)
	}
}

func TestMoveToActiveLoadoutFull(t *testing.T) {
	loadout, _ := NewLoadout([]string{"a", "b", "c", "d", "e"}, []string{"f"})
	_, err := loadout.MoveToActive("f")
	if err != ErrLoadoutFull {
		t.Fatalf("expected ErrLoadoutFull, got %v", err)
	}
}

func TestMoveToVaultCardNotFound(t *testing.T) {
	loadout, _ := NewLoadout([]string{"a"}, []string{})
	_, err := loadout.MoveToVault("missing")
	if err != ErrCardNotFound {
		t.Fatalf("expected ErrCardNotFound, got %v", err)
	}
}

func TestNewLoadoutDuplicateInActive(t *testing.T) {
	_, err := NewLoadout([]string{"a", "a"}, nil)
	if err != ErrDuplicateCard {
		t.Fatalf("expected ErrDuplicateCard, got %v", err)
	}
}

func TestNewLoadoutValid(t *testing.T) {
	l, err := NewLoadout([]string{"a", "b"}, []string{"c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(l.Active) != 2 || len(l.Vault) != 1 {
		t.Fatalf("unexpected loadout: %+v", l)
	}
}

func TestMoveToActiveWithRecallInsufficientStress(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		Hope:        2,
		HopeMax:     HopeMax,
		Stress:      0,
		StressMax:   6,
		Armor:       0,
		ArmorMax:    0,
	})
	loadout, err := NewLoadout([]string{}, []string{"card-1"})
	if err != nil {
		t.Fatalf("NewLoadout returned error: %v", err)
	}
	_, err = loadout.MoveToActiveWithRecall(DomainCard{ID: "card-1", RecallCost: 3}, state, false)
	if err == nil {
		t.Fatal("expected error when stress is insufficient for recall cost")
	}
}

func TestMoveToActiveWithRecallZeroCost(t *testing.T) {
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
	updated, err := loadout.MoveToActiveWithRecall(DomainCard{ID: "card-1", RecallCost: 0}, state, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Stress() != 2 {
		t.Fatalf("stress = %d, want 2 (no cost for zero recall)", state.Stress())
	}
	if len(updated.Active) != 1 {
		t.Fatal("expected card in active loadout")
	}
}
