package mechanics

import "testing"

func TestNewCharacterState_ClampsAndDefaults(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          100,
		HPMax:       0,
		Hope:        -1,
		HopeMax:     0,
		Stress:      100,
		StressMax:   0,
		Armor:       -10,
		ArmorMax:    -1,
		LifeState:   "",
	})
	if state.HP != HPMaxDefault {
		t.Fatalf("HP = %d, want %d", state.HP, HPMaxDefault)
	}
	if state.Hope != HopeMin || state.HopeMax != HopeMaxDefault {
		t.Fatalf("unexpected hope values: hope=%d hopeMax=%d", state.Hope, state.HopeMax)
	}
	if state.Stress != StressMaxDefault || state.StressMax != StressMaxDefault {
		t.Fatalf("unexpected stress values: stress=%d stressMax=%d", state.Stress, state.StressMax)
	}
	if state.Armor != ArmorMin || state.ArmorMax != ArmorMin {
		t.Fatalf("unexpected armor values: armor=%d armorMax=%d", state.Armor, state.ArmorMax)
	}
	if state.LifeState != LifeStateAlive {
		t.Fatalf("LifeState = %q, want %q", state.LifeState, LifeStateAlive)
	}
}

func TestApplyTemporaryArmor_ReplacesSameSourceBucket(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       0,
		ArmorMax:    2,
	})
	state.ApplyTemporaryArmor(TemporaryArmorBucket{Source: "spell", Duration: "short_rest", SourceID: "s1", Amount: 2})
	state.ApplyTemporaryArmor(TemporaryArmorBucket{Source: "spell", Duration: "short_rest", SourceID: "s1", Amount: 1})
	if got := state.TemporaryArmorAmount(); got != 1 {
		t.Fatalf("TemporaryArmorAmount = %d, want 1", got)
	}
	if state.Armor != 1 {
		t.Fatalf("Armor = %d, want 1", state.Armor)
	}
}

func TestGainStress_OverflowDamagesHP(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		Stress:      5,
		StressMax:   6,
	})
	result, err := state.GainStress(3)
	if err != nil {
		t.Fatalf("GainStress: %v", err)
	}
	if !result.LastStressMarked {
		t.Fatal("expected LastStressMarked=true")
	}
	if result.Overflow != 2 || state.HP != 4 || state.Stress != 6 {
		t.Fatalf("unexpected overflow result: %+v state={HP:%d Stress:%d}", result, state.HP, state.Stress)
	}
}

func TestClearTemporaryArmorByDuration(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       0,
		ArmorMax:    2,
	})
	state.ApplyTemporaryArmor(TemporaryArmorBucket{Source: "spell", Duration: "short_rest", Amount: 2})
	state.ApplyTemporaryArmor(TemporaryArmorBucket{Source: "ritual", Duration: "long_rest", Amount: 1})
	removed := state.ClearTemporaryArmorByDuration("short_rest")
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if got := state.TemporaryArmorAmount(); got != 1 {
		t.Fatalf("TemporaryArmorAmount = %d, want 1", got)
	}
}
