package reducer

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
)

func TestApplyCharacterStatePatchAndNormalize(t *testing.T) {
	state := &mechanics.CharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          3,
		Hope:        1,
		HopeMax:     0,
		Stress:      1,
		Armor:       0,
		LifeState:   "",
	}
	hpAfter := 5
	lifeState := mechanics.LifeStateUnconscious
	ApplyCharacterStatePatch(state, CharacterStatePatch{
		HPAfter:        &hpAfter,
		LifeStateAfter: &lifeState,
	})
	if err := NormalizeAndValidateCharacterState(state); err != nil {
		t.Fatalf("NormalizeAndValidateCharacterState: %v", err)
	}
	if state.HP != 5 {
		t.Fatalf("HP = %d, want 5", state.HP)
	}
	if state.HopeMax != mechanics.HopeMax {
		t.Fatalf("HopeMax = %d, want %d", state.HopeMax, mechanics.HopeMax)
	}
	if state.LifeState != mechanics.LifeStateUnconscious {
		t.Fatalf("LifeState = %q, want %q", state.LifeState, mechanics.LifeStateUnconscious)
	}
}

func TestApplyDowntimeMove_RepairAllArmor(t *testing.T) {
	state := mechanics.NewCharacterState(mechanics.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       0,
		ArmorMax:    2,
	})
	state.ApplyTemporaryArmor(mechanics.TemporaryArmorBucket{Source: "spell", Duration: "short_rest", Amount: 2})
	state.ApplyTemporaryArmor(mechanics.TemporaryArmorBucket{Source: "ritual", Duration: "long_rest", Amount: 1})
	ApplyDowntimeMove(state, "repair_all_armor", nil, nil, nil)
	if state.Armor != state.ResourceCap(mechanics.ResourceArmor) {
		t.Fatalf("Armor = %d, want cap %d", state.Armor, state.ResourceCap(mechanics.ResourceArmor))
	}
	if state.TemporaryArmorAmount() != 1 {
		t.Fatalf("TemporaryArmorAmount = %d, want 1", state.TemporaryArmorAmount())
	}
}

func TestNormalizeAndValidateCharacterState_RejectsInvalidRange(t *testing.T) {
	state := &mechanics.CharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          mechanics.HPMaxCap + 1,
		Hope:        1,
		HopeMax:     mechanics.HopeMax,
		Stress:      0,
		Armor:       0,
		LifeState:   mechanics.LifeStateAlive,
	}
	if err := NormalizeAndValidateCharacterState(state); err == nil {
		t.Fatal("expected validation error for HP out of range")
	}
}

func TestApplyCharacterStatePatch_AllFields(t *testing.T) {
	state := mechanics.NewCharacterState(mechanics.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          2,
		HPMax:       6,
		Hope:        1,
		HopeMax:     6,
		Stress:      2,
		StressMax:   6,
		Armor:       1,
		ArmorMax:    2,
	})

	hp := 5
	hope := 3
	hopeMax := 5
	stress := 4
	armor := 0
	life := mechanics.LifeStateUnconscious
	ApplyCharacterStatePatch(state, CharacterStatePatch{
		HPAfter:        &hp,
		HopeAfter:      &hope,
		HopeMaxAfter:   &hopeMax,
		StressAfter:    &stress,
		ArmorAfter:     &armor,
		LifeStateAfter: &life,
	})

	if state.HP != 5 || state.Hope != 3 || state.HopeMax != 5 || state.Stress != 4 || state.Armor != 0 || state.LifeState != mechanics.LifeStateUnconscious {
		t.Fatalf("unexpected patched state: %+v", *state)
	}
}

func TestApplyConditionPatchCopiesSlice(t *testing.T) {
	state := mechanics.NewCharacterState(mechanics.CharacterStateConfig{CampaignID: "camp-1", CharacterID: "char-1"})
	conditions := []string{"hidden", "vulnerable"}
	ApplyConditionPatch(state, conditions)
	conditions[0] = "changed"
	if state.Conditions[0] != "hidden" {
		t.Fatalf("conditions should be copied, got %v", state.Conditions)
	}
}

func TestApplyLoadoutSwapNilAndSet(t *testing.T) {
	ApplyLoadoutSwap(nil, intPtr(2))

	state := mechanics.NewCharacterState(mechanics.CharacterStateConfig{CampaignID: "camp-1", CharacterID: "char-1", Stress: 1, StressMax: 6})
	ApplyLoadoutSwap(state, nil)
	if state.Stress != 1 {
		t.Fatalf("stress should be unchanged when stressAfter=nil, got %d", state.Stress)
	}
	ApplyLoadoutSwap(state, intPtr(3))
	if state.Stress != 3 {
		t.Fatalf("stress = %d, want 3", state.Stress)
	}
}

func TestApplyTemporaryArmorSetsDefaultLifeState(t *testing.T) {
	ApplyTemporaryArmor(nil, TemporaryArmorPatch{Source: "x", Duration: "short_rest", Amount: 1})

	state := mechanics.NewCharacterState(mechanics.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		LifeState:   "",
		Armor:       0,
		ArmorMax:    2,
	})
	ApplyTemporaryArmor(state, TemporaryArmorPatch{
		Source:   "  spell  ",
		Duration: "  short_rest  ",
		SourceID: "  s1  ",
		Amount:   1,
	})
	if state.LifeState != mechanics.LifeStateAlive {
		t.Fatalf("LifeState = %q, want %q", state.LifeState, mechanics.LifeStateAlive)
	}
	if len(state.ArmorBonus) != 1 {
		t.Fatalf("ArmorBonus len = %d, want 1", len(state.ArmorBonus))
	}
	if state.ArmorBonus[0].Source != "spell" || state.ArmorBonus[0].Duration != "short_rest" || state.ArmorBonus[0].SourceID != "s1" {
		t.Fatalf("temporary armor should be normalized, got %+v", state.ArmorBonus[0])
	}
}

func TestApplyRestPatchUpdatesProvidedFields(t *testing.T) {
	ApplyRestPatch(nil, RestCharacterPatch{HopeAfter: intPtr(1)})

	state := mechanics.NewCharacterState(mechanics.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hope:        1,
		HopeMax:     6,
		Stress:      3,
		StressMax:   6,
		Armor:       2,
		ArmorMax:    2,
	})
	ApplyRestPatch(state, RestCharacterPatch{HopeAfter: intPtr(4), StressAfter: intPtr(1)})
	if state.Hope != 4 || state.Stress != 1 || state.Armor != 2 {
		t.Fatalf("unexpected rest patch result: %+v", *state)
	}
}

func TestClearRestTemporaryArmorClearsSelectedDurations(t *testing.T) {
	ClearRestTemporaryArmor(nil, true, true)

	state := mechanics.NewCharacterState(mechanics.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       0,
		ArmorMax:    2,
	})
	state.ApplyTemporaryArmor(mechanics.TemporaryArmorBucket{Source: "spell", Duration: "short_rest", Amount: 2})
	state.ApplyTemporaryArmor(mechanics.TemporaryArmorBucket{Source: "ritual", Duration: "long_rest", Amount: 1})

	ClearRestTemporaryArmor(state, true, false)
	if state.TemporaryArmorAmount() != 1 {
		t.Fatalf("TemporaryArmorAmount = %d, want 1", state.TemporaryArmorAmount())
	}
	if state.Armor != 3 {
		t.Fatalf("Armor = %d, want 3 (base cap + long rest temp)", state.Armor)
	}

	ClearRestTemporaryArmor(state, false, true)
	if state.TemporaryArmorAmount() != 0 {
		t.Fatalf("TemporaryArmorAmount = %d, want 0", state.TemporaryArmorAmount())
	}
	if state.Armor != 2 {
		t.Fatalf("Armor = %d, want base cap 2", state.Armor)
	}
}

func TestApplyDamageUpdatesProvidedFields(t *testing.T) {
	ApplyDamage(nil, intPtr(2), intPtr(1))

	state := mechanics.NewCharacterState(mechanics.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		Armor:       1,
		ArmorMax:    2,
	})
	ApplyDamage(state, intPtr(4), nil)
	if state.HP != 4 || state.Armor != 1 {
		t.Fatalf("unexpected damage patch result: %+v", *state)
	}
	ApplyDamage(state, nil, intPtr(0))
	if state.Armor != 0 {
		t.Fatalf("Armor = %d, want 0", state.Armor)
	}
}

func TestApplyDowntimeMoveBranches(t *testing.T) {
	ApplyDowntimeMove(nil, "repair_all_armor", intPtr(2), intPtr(1), intPtr(0))

	state := mechanics.NewCharacterState(mechanics.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hope:        1,
		HopeMax:     6,
		Stress:      2,
		StressMax:   6,
		Armor:       0,
		ArmorMax:    2,
	})
	state.ApplyTemporaryArmor(mechanics.TemporaryArmorBucket{Source: "spell", Duration: "short_rest", Amount: 2})
	ApplyDowntimeMove(state, "prepare", intPtr(3), intPtr(1), nil)
	if state.Hope != 3 || state.Stress != 1 || state.TemporaryArmorAmount() != 2 {
		t.Fatalf("prepare branch should preserve temp armor, got %+v", *state)
	}

	ApplyDowntimeMove(state, "repair_all_armor", nil, nil, nil)
	if state.TemporaryArmorAmount() != 0 {
		t.Fatalf("repair_all_armor should clear short-rest temp armor, got %d", state.TemporaryArmorAmount())
	}
	if state.Armor != state.ResourceCap(mechanics.ResourceArmor) {
		t.Fatalf("Armor = %d, want cap %d", state.Armor, state.ResourceCap(mechanics.ResourceArmor))
	}
}

func TestNormalizeAndValidateCharacterState_Branches(t *testing.T) {
	t.Run("nil state is valid", func(t *testing.T) {
		if err := NormalizeAndValidateCharacterState(nil); err != nil {
			t.Fatalf("expected nil state to be valid, got %v", err)
		}
	})

	t.Run("defaults hope max and life state", func(t *testing.T) {
		state := &mechanics.CharacterState{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			HP:          1,
			Hope:        1,
			HopeMax:     0,
			Stress:      0,
			Armor:       0,
			LifeState:   "",
		}
		if err := NormalizeAndValidateCharacterState(state); err != nil {
			t.Fatalf("NormalizeAndValidateCharacterState: %v", err)
		}
		if state.HopeMax != mechanics.HopeMax {
			t.Fatalf("HopeMax = %d, want %d", state.HopeMax, mechanics.HopeMax)
		}
		if state.LifeState != mechanics.LifeStateAlive {
			t.Fatalf("LifeState = %q, want %q", state.LifeState, mechanics.LifeStateAlive)
		}
	})

	t.Run("rejects invalid ranges and life state", func(t *testing.T) {
		tests := []mechanics.CharacterState{
			{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				HP:          1,
				Hope:        1,
				HopeMax:     mechanics.HopeMax + 1,
				Stress:      0,
				Armor:       0,
				LifeState:   mechanics.LifeStateAlive,
			},
			{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				HP:          1,
				Hope:        mechanics.HopeMax + 1,
				HopeMax:     mechanics.HopeMax,
				Stress:      0,
				Armor:       0,
				LifeState:   mechanics.LifeStateAlive,
			},
			{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				HP:          1,
				Hope:        1,
				HopeMax:     mechanics.HopeMax,
				Stress:      mechanics.StressMaxCap + 1,
				Armor:       0,
				LifeState:   mechanics.LifeStateAlive,
			},
			{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				HP:          1,
				Hope:        1,
				HopeMax:     mechanics.HopeMax,
				Stress:      0,
				Armor:       mechanics.ArmorMaxCap + 1,
				LifeState:   mechanics.LifeStateAlive,
			},
			{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				HP:          1,
				Hope:        1,
				HopeMax:     mechanics.HopeMax,
				Stress:      0,
				Armor:       0,
				LifeState:   "unsupported",
			},
		}
		for _, tc := range tests {
			state := tc
			if err := NormalizeAndValidateCharacterState(&state); err == nil {
				t.Fatalf("expected validation error for state: %+v", tc)
			}
		}
	})
}

func intPtr(v int) *int {
	return &v
}
