package daggerheart

import "testing"

func TestNormalizedDiceValues(t *testing.T) {
	// nil input returns nil.
	if got := normalizedDiceValues(nil); got != nil {
		t.Fatalf("normalizedDiceValues(nil) = %v, want nil", got)
	}
	// All zero/negative returns nil.
	if got := normalizedDiceValues([]int{0, -1, -2}); got != nil {
		t.Fatalf("normalizedDiceValues(all invalid) = %v, want nil", got)
	}
	// Mixed input filters out non-positive values.
	got := normalizedDiceValues([]int{6, 0, 8, -1, 12})
	if len(got) != 3 || got[0] != 6 || got[1] != 8 || got[2] != 12 {
		t.Fatalf("normalizedDiceValues(mixed) = %v, want [6 8 12]", got)
	}
}

func TestCharacterClassState_IsZero(t *testing.T) {
	if !(CharacterClassState{}).IsZero() {
		t.Fatal("zero-value class state should be IsZero")
	}
	if (CharacterClassState{AttackBonusUntilRest: 1}).IsZero() {
		t.Fatal("state with attack bonus should not be IsZero")
	}
	if (CharacterClassState{RallyDice: []int{6}}).IsZero() {
		t.Fatal("state with rally dice should not be IsZero")
	}
	if (CharacterClassState{PrayerDice: []int{8}}).IsZero() {
		t.Fatal("state with prayer dice should not be IsZero")
	}
	if (CharacterClassState{Unstoppable: CharacterUnstoppableState{Active: true}}).IsZero() {
		t.Fatal("state with unstoppable active should not be IsZero")
	}
	if (CharacterClassState{ChannelRawPowerUsedThisLongRest: true}).IsZero() {
		t.Fatal("state with channel raw power used should not be IsZero")
	}
}

func TestCharacterClassState_Normalized_ClampsNegatives(t *testing.T) {
	state := CharacterClassState{
		AttackBonusUntilRest:       -1,
		EvasionBonusUntilHitOrRest: -2,
		DifficultyPenaltyUntilRest: 3, // positive should be clamped to 0
		StrangePatternsNumber:      -1,
		Unstoppable: CharacterUnstoppableState{
			CurrentValue: -5,
			DieSides:     -4,
		},
	}
	got := state.Normalized()
	if got.AttackBonusUntilRest != 0 {
		t.Fatalf("attack bonus = %d, want 0", got.AttackBonusUntilRest)
	}
	if got.EvasionBonusUntilHitOrRest != 0 {
		t.Fatalf("evasion bonus = %d, want 0", got.EvasionBonusUntilHitOrRest)
	}
	if got.DifficultyPenaltyUntilRest != 0 {
		t.Fatalf("difficulty penalty = %d, want 0", got.DifficultyPenaltyUntilRest)
	}
	if got.StrangePatternsNumber != 0 {
		t.Fatalf("strange patterns = %d, want 0", got.StrangePatternsNumber)
	}
	if got.Unstoppable.CurrentValue != 0 {
		t.Fatalf("unstoppable current = %d, want 0", got.Unstoppable.CurrentValue)
	}
	if got.Unstoppable.DieSides != 0 {
		t.Fatalf("unstoppable die sides = %d, want 0", got.Unstoppable.DieSides)
	}
}

func TestDerefInt(t *testing.T) {
	if got := derefInt(nil, 42); got != 42 {
		t.Fatalf("derefInt(nil, 42) = %d, want 42", got)
	}
	v := 7
	if got := derefInt(&v, 42); got != 7 {
		t.Fatalf("derefInt(&7, 42) = %d, want 7", got)
	}
}

func TestWithActiveBeastform(t *testing.T) {
	state := CharacterClassState{}
	active := &CharacterActiveBeastformState{
		BeastformID: "bf-1",
		BaseTrait:   "strength",
		AttackTrait: "strength",
		TraitBonus:  2,
		DamageDice:  []CharacterDamageDie{{Count: 1, Sides: 8}},
		DamageType:  "physical",
	}
	got := WithActiveBeastform(state, active)
	if got.ActiveBeastform == nil {
		t.Fatal("expected active beastform to be set")
	}
	if got.ActiveBeastform.BeastformID != "bf-1" {
		t.Fatalf("beastform id = %q, want %q", got.ActiveBeastform.BeastformID, "bf-1")
	}

	// Nil beastform clears active.
	got = WithActiveBeastform(got, nil)
	if got.ActiveBeastform != nil {
		t.Fatal("expected active beastform to be nil after clearing")
	}
}

func TestNormalizedActiveBeastform_EmptyID(t *testing.T) {
	// Empty beastform ID normalizes to nil.
	got := normalizedActiveBeastformPtr(&CharacterActiveBeastformState{BeastformID: "  "})
	if got != nil {
		t.Fatal("expected nil for empty beastform ID")
	}
}
