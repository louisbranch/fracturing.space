package mechanics

import (
	"testing"

	domainerrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
)

func TestCharacterState_IDAccessorsAndHPHelpers(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          3,
		HPMax:       6,
	})

	if got := state.CampaignIDValue(); got != "camp-1" {
		t.Fatalf("CampaignIDValue = %q, want camp-1", got)
	}
	if got := state.CharacterIDValue(); got != "char-1" {
		t.Fatalf("CharacterIDValue = %q, want char-1", got)
	}
	if got := state.MaxHP(); got != 6 {
		t.Fatalf("MaxHP = %d, want 6", got)
	}
	if got := state.CurrentHP(); got != 3 {
		t.Fatalf("CurrentHP = %d, want 3", got)
	}

	before, after := state.Heal(10)
	if before != 3 || after != 6 {
		t.Fatalf("Heal before/after = %d/%d, want 3/6", before, after)
	}
	before, after = state.TakeDamage(4)
	if before != 6 || after != 2 {
		t.Fatalf("TakeDamage before/after = %d/%d, want 6/2", before, after)
	}
}

func TestCharacterState_GainResourceVariants(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		Hope:        1,
		HopeMax:     3,
		Stress:      1,
		StressMax:   3,
		Armor:       1,
		ArmorMax:    2,
	})

	before, after, err := state.GainResource(ResourceHope, 2)
	if err != nil {
		t.Fatalf("GainResource hope: %v", err)
	}
	if before != 1 || after != 3 {
		t.Fatalf("GainResource hope before/after = %d/%d, want 1/3", before, after)
	}

	before, after, err = state.GainResource(ResourceStress, 2)
	if err != nil {
		t.Fatalf("GainResource stress: %v", err)
	}
	if before != 1 || after != 3 {
		t.Fatalf("GainResource stress before/after = %d/%d, want 1/3", before, after)
	}

	before, after, err = state.GainResource(ResourceArmor, 10)
	if err != nil {
		t.Fatalf("GainResource armor: %v", err)
	}
	if before != 1 || after != 2 {
		t.Fatalf("GainResource armor before/after = %d/%d, want 1/2", before, after)
	}

	before, after, err = state.GainResource(ResourceHope, -5)
	if err != nil {
		t.Fatalf("GainResource negative hope: %v", err)
	}
	if before != 3 || after != 3 {
		t.Fatalf("GainResource negative hope before/after = %d/%d, want 3/3", before, after)
	}

	_, _, err = state.GainResource("mystery", 1)
	if domainerrors.GetCode(err) != domainerrors.CodeDaggerheartUnknownResource {
		t.Fatalf("unknown resource code = %s, want %s", domainerrors.GetCode(err), domainerrors.CodeDaggerheartUnknownResource)
	}
}

func TestCharacterState_SpendResourceVariants(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hope:        2,
		HopeMax:     3,
		Stress:      2,
		StressMax:   3,
		Armor:       2,
		ArmorMax:    2,
	})

	before, after, err := state.SpendResource(ResourceHope, 1)
	if err != nil {
		t.Fatalf("SpendResource hope: %v", err)
	}
	if before != 2 || after != 1 {
		t.Fatalf("SpendResource hope before/after = %d/%d, want 2/1", before, after)
	}

	before, after, err = state.SpendResource(ResourceArmor, -2)
	if err != nil {
		t.Fatalf("SpendResource armor negative amount: %v", err)
	}
	if before != 2 || after != 2 {
		t.Fatalf("SpendResource armor negative before/after = %d/%d, want 2/2", before, after)
	}

	_, _, err = state.SpendResource(ResourceStress, 5)
	if domainerrors.GetCode(err) != domainerrors.CodeDaggerheartInsufficientResource {
		t.Fatalf("insufficient resource code = %s, want %s", domainerrors.GetCode(err), domainerrors.CodeDaggerheartInsufficientResource)
	}

	_, _, err = state.SpendResource("mystery", 1)
	if domainerrors.GetCode(err) != domainerrors.CodeDaggerheartUnknownResource {
		t.Fatalf("unknown resource code = %s, want %s", domainerrors.GetCode(err), domainerrors.CodeDaggerheartUnknownResource)
	}
}

func TestCharacterState_GainStressBranches(t *testing.T) {
	t.Run("no-op amount", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			HP:          6,
			HPMax:       6,
			Stress:      1,
			StressMax:   3,
		})
		result, err := state.GainStress(0)
		if err != nil {
			t.Fatalf("GainStress: %v", err)
		}
		if result.StressAfter != 1 || result.HPAfter != 6 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("already at cap converts all to hp damage", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			HP:          5,
			HPMax:       6,
			Stress:      3,
			StressMax:   3,
		})
		result, err := state.GainStress(2)
		if err != nil {
			t.Fatalf("GainStress: %v", err)
		}
		if result.Overflow != 2 || result.StressAfter != 3 || result.HPAfter != 3 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("partial gain without overflow", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			HP:          6,
			HPMax:       6,
			Stress:      1,
			StressMax:   4,
		})
		result, err := state.GainStress(2)
		if err != nil {
			t.Fatalf("GainStress: %v", err)
		}
		if result.StressAfter != 3 || result.HPAfter != 6 || result.LastStressMarked {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("exactly marks last stress without overflow", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			HP:          6,
			HPMax:       6,
			Stress:      2,
			StressMax:   4,
		})
		result, err := state.GainStress(2)
		if err != nil {
			t.Fatalf("GainStress: %v", err)
		}
		if !result.LastStressMarked || result.Overflow != 0 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})
}

func TestCharacterState_ResourceViewsAndSetters(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hope:        2,
		HopeMax:     4,
		Stress:      1,
		StressMax:   3,
		Armor:       0,
		ArmorMax:    2,
	})
	state.ApplyTemporaryArmor(TemporaryArmorBucket{Source: "spell", Duration: "short_rest", Amount: 1})

	if got := state.ResourceValue(ResourceHope); got != 2 {
		t.Fatalf("ResourceValue hope = %d, want 2", got)
	}
	if got := state.ResourceValue(ResourceStress); got != 1 {
		t.Fatalf("ResourceValue stress = %d, want 1", got)
	}
	if got := state.ResourceValue(ResourceArmor); got != 1 {
		t.Fatalf("ResourceValue armor = %d, want 1", got)
	}
	if got := state.ResourceValue("unknown"); got != 0 {
		t.Fatalf("ResourceValue unknown = %d, want 0", got)
	}

	if got := state.ResourceCap(ResourceHope); got != 4 {
		t.Fatalf("ResourceCap hope = %d, want 4", got)
	}
	if got := state.ResourceCap(ResourceStress); got != 3 {
		t.Fatalf("ResourceCap stress = %d, want 3", got)
	}
	if got := state.ResourceCap(ResourceArmor); got != 3 {
		t.Fatalf("ResourceCap armor = %d, want 3", got)
	}
	if got := state.ResourceCap("unknown"); got != 0 {
		t.Fatalf("ResourceCap unknown = %d, want 0", got)
	}

	names := state.ResourceNames()
	if len(names) != 3 || names[0] != ResourceHope || names[1] != ResourceStress || names[2] != ResourceArmor {
		t.Fatalf("ResourceNames = %v, want [hope stress armor]", names)
	}

	state.SetHope(10)
	if state.Hope != 4 {
		t.Fatalf("SetHope should clamp to 4, got %d", state.Hope)
	}
	state.SetHopeMax(2)
	if state.HopeMax != 2 || state.Hope != 2 {
		t.Fatalf("SetHopeMax should clamp both hope_max and hope, got hope=%d hope_max=%d", state.Hope, state.HopeMax)
	}
	state.SetStress(10)
	if state.Stress != 3 {
		t.Fatalf("SetStress should clamp to 3, got %d", state.Stress)
	}
	state.SetArmor(10)
	if state.Armor != 3 {
		t.Fatalf("SetArmor should clamp to armor cap 3, got %d", state.Armor)
	}
}

func TestCharacterState_ApplyTemporaryArmorEdgeCases(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       1,
		ArmorMax:    5,
	})

	state.ApplyTemporaryArmor(TemporaryArmorBucket{Source: "", Duration: "short_rest", Amount: 1})
	if got := len(state.ArmorBonus); got != 0 {
		t.Fatalf("invalid bucket should not be added, got %d buckets", got)
	}

	state.ArmorBonus = []TemporaryArmorBucket{
		{
			Source:   "spell",
			Duration: "short_rest",
			Amount:   -2,
		},
	}
	state.ApplyTemporaryArmor(TemporaryArmorBucket{
		Source:   "spell",
		Duration: "short_rest",
		Amount:   2,
	})
	if got := state.Armor; got != 3 {
		t.Fatalf("Armor = %d, want 3", got)
	}
	if got := state.ArmorBonus[0].Amount; got != 2 {
		t.Fatalf("replaced bucket amount = %d, want 2", got)
	}
}

func TestCharacterState_ClearTemporaryArmorByDurationNoMatch(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       0,
		ArmorMax:    3,
	})
	state.ApplyTemporaryArmor(TemporaryArmorBucket{Source: "spell", Duration: "short_rest", Amount: 1})

	removed := state.ClearTemporaryArmorByDuration("long_rest")
	if removed != 0 {
		t.Fatalf("removed = %d, want 0", removed)
	}
	if got := state.TemporaryArmorAmount(); got != 1 {
		t.Fatalf("TemporaryArmorAmount = %d, want 1", got)
	}
}
