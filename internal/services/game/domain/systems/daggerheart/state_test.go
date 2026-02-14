package daggerheart

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

func TestCharacterState_ResourceHolder(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          10,
		HPMax:       10,
		Hope:        2,
		HopeMax:     HopeMax,
		Stress:      0,
		StressMax:   6,
		Armor:       1,
		ArmorMax:    2,
	})

	// Test GainResource for Hope
	t.Run("GainHope", func(t *testing.T) {
		before, after, err := state.GainResource(ResourceHope, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if before != 2 {
			t.Errorf("before = %d, want 2", before)
		}
		if after != 4 {
			t.Errorf("after = %d, want 4", after)
		}
	})

	// Test Hope cap
	t.Run("HopeCap", func(t *testing.T) {
		before, after, err := state.GainResource(ResourceHope, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if after != HopeMax {
			t.Errorf("after = %d, want %d (capped)", after, HopeMax)
		}
		_ = before
	})

	// Reset for stress tests
	state.SetHope(2)

	// Test SpendResource for Hope
	t.Run("SpendHope", func(t *testing.T) {
		before, after, err := state.SpendResource(ResourceHope, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if before != 2 {
			t.Errorf("before = %d, want 2", before)
		}
		if after != 1 {
			t.Errorf("after = %d, want 1", after)
		}
	})

	// Test insufficient Hope
	t.Run("InsufficientHope", func(t *testing.T) {
		_, _, err := state.SpendResource(ResourceHope, 10)
		if err == nil {
			t.Fatal("expected error for insufficient hope")
		}
	})

	// Test GainResource for Stress
	t.Run("GainStress", func(t *testing.T) {
		before, after, err := state.GainResource(ResourceStress, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if before != 0 {
			t.Errorf("before = %d, want 0", before)
		}
		if after != 2 {
			t.Errorf("after = %d, want 2", after)
		}
	})

	// Test stress overflow to HP
	t.Run("StressOverflow", func(t *testing.T) {
		state.SetStress(5)
		beforeHP := state.CurrentHP()
		before, after, err := state.GainResource(ResourceStress, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if before != 5 {
			t.Errorf("before = %d, want 5", before)
		}
		if after != 6 {
			t.Errorf("after = %d, want 6", after)
		}
		if state.CurrentHP() != beforeHP-1 {
			t.Errorf("hp = %d, want %d", state.CurrentHP(), beforeHP-1)
		}
	})

	// Test unknown resource
	t.Run("UnknownResource", func(t *testing.T) {
		_, _, err := state.GainResource("unknown", 1)
		if err == nil {
			t.Fatal("expected error for unknown resource")
		}
	})

	// Test ResourceValue
	t.Run("ResourceValue", func(t *testing.T) {
		state.SetStress(2)
		if v := state.ResourceValue(ResourceHope); v != 1 {
			t.Errorf("hope = %d, want 1", v)
		}
		if v := state.ResourceValue(ResourceStress); v != 2 {
			t.Errorf("stress = %d, want 2", v)
		}
		if v := state.ResourceValue(ResourceArmor); v != 1 {
			t.Errorf("armor = %d, want 1", v)
		}
		if v := state.ResourceValue("unknown"); v != 0 {
			t.Errorf("unknown = %d, want 0", v)
		}
	})

	// Test ResourceCap
	t.Run("ResourceCap", func(t *testing.T) {
		if v := state.ResourceCap(ResourceHope); v != HopeMax {
			t.Errorf("hope cap = %d, want %d", v, HopeMax)
		}
		if v := state.ResourceCap(ResourceStress); v != 6 {
			t.Errorf("stress cap = %d, want 6", v)
		}
		if v := state.ResourceCap(ResourceArmor); v != 2 {
			t.Errorf("armor cap = %d, want 2", v)
		}
	})

	// Test ResourceNames
	t.Run("ResourceNames", func(t *testing.T) {
		names := state.ResourceNames()
		if len(names) != 3 {
			t.Fatalf("len(names) = %d, want 3", len(names))
		}
		hasHope := false
		hasStress := false
		hasArmor := false
		for _, name := range names {
			if name == ResourceHope {
				hasHope = true
			}
			if name == ResourceStress {
				hasStress = true
			}
			if name == ResourceArmor {
				hasArmor = true
			}
		}
		if !hasHope || !hasStress || !hasArmor {
			t.Errorf("expected hope, stress, armor in names, got %v", names)
		}
	})
}

func TestCharacterState_ClampsConfig(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          20,
		HPMax:       20,
		Hope:        10,
		HopeMax:     HopeMax,
		Stress:      20,
		StressMax:   20,
		Armor:       5,
		ArmorMax:    20,
	})

	if v := state.MaxHP(); v != HPMaxCap {
		t.Fatalf("MaxHP = %d, want %d", v, HPMaxCap)
	}
	if v := state.CurrentHP(); v != HPMaxCap {
		t.Fatalf("CurrentHP = %d, want %d", v, HPMaxCap)
	}
	if v := state.ResourceCap(ResourceStress); v != StressMaxCap {
		t.Fatalf("Stress cap = %d, want %d", v, StressMaxCap)
	}
	if v := state.ResourceValue(ResourceStress); v != StressMaxCap {
		t.Fatalf("Stress = %d, want %d", v, StressMaxCap)
	}
	if v := state.ResourceCap(ResourceArmor); v != ArmorMaxCap {
		t.Fatalf("Armor cap = %d, want %d", v, ArmorMaxCap)
	}
	if v := state.ResourceValue(ResourceArmor); v != 5 {
		t.Fatalf("Armor = %d, want %d", v, 5)
	}
	if v := state.ResourceValue(ResourceHope); v != HopeMax {
		t.Fatalf("Hope = %d, want %d", v, HopeMax)
	}
}

func TestCharacterState_Healable(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          5,
		HPMax:       10,
		Hope:        2,
		HopeMax:     HopeMax,
		Stress:      0,
		StressMax:   6,
	})

	t.Run("Heal", func(t *testing.T) {
		before, after := state.Heal(3)
		if before != 5 {
			t.Errorf("before = %d, want 5", before)
		}
		if after != 8 {
			t.Errorf("after = %d, want 8", after)
		}
	})

	t.Run("HealCapped", func(t *testing.T) {
		before, after := state.Heal(10)
		if after != 10 {
			t.Errorf("after = %d, want 10 (capped at max)", after)
		}
		_ = before
	})

	t.Run("MaxHP", func(t *testing.T) {
		if v := state.MaxHP(); v != 10 {
			t.Errorf("MaxHP = %d, want 10", v)
		}
	})
}

func TestCharacterState_Damageable(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          10,
		HPMax:       10,
		Hope:        2,
		HopeMax:     HopeMax,
		Stress:      0,
		StressMax:   6,
	})

	t.Run("TakeDamage", func(t *testing.T) {
		before, after := state.TakeDamage(3)
		if before != 10 {
			t.Errorf("before = %d, want 10", before)
		}
		if after != 7 {
			t.Errorf("after = %d, want 7", after)
		}
	})

	t.Run("TakeDamageFloor", func(t *testing.T) {
		before, after := state.TakeDamage(100)
		if after != 0 {
			t.Errorf("after = %d, want 0 (floored)", after)
		}
		_ = before
	})

	t.Run("CurrentHP", func(t *testing.T) {
		if v := state.CurrentHP(); v != 0 {
			t.Errorf("CurrentHP = %d, want 0", v)
		}
	})
}

func TestSnapshotState_ResourceHolder(t *testing.T) {
	ss := NewSnapshotState(SnapshotStateConfig{
		CampaignID: "camp-1",
		GMFear:     5,
	})

	t.Run("GainGMFear", func(t *testing.T) {
		before, after, err := ss.GainResource(ResourceGMFear, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if before != 5 {
			t.Errorf("before = %d, want 5", before)
		}
		if after != 8 {
			t.Errorf("after = %d, want 8", after)
		}
	})

	t.Run("GMFearCap", func(t *testing.T) {
		before, after, err := ss.GainResource(ResourceGMFear, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if after != GMFearMax {
			t.Errorf("after = %d, want %d (capped)", after, GMFearMax)
		}
		_ = before
	})

	// Reset
	ss.SetGMFear(5)

	t.Run("SpendGMFear", func(t *testing.T) {
		before, after, err := ss.SpendResource(ResourceGMFear, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if before != 5 {
			t.Errorf("before = %d, want 5", before)
		}
		if after != 3 {
			t.Errorf("after = %d, want 3", after)
		}
	})

	t.Run("InsufficientGMFear", func(t *testing.T) {
		_, _, err := ss.SpendResource(ResourceGMFear, 100)
		if err == nil {
			t.Fatal("expected error for insufficient GM Fear")
		}
	})

	t.Run("UnknownResource", func(t *testing.T) {
		_, _, err := ss.GainResource("unknown", 1)
		if err == nil {
			t.Fatal("expected error for unknown resource")
		}
	})

	t.Run("ResourceValue", func(t *testing.T) {
		if v := ss.ResourceValue(ResourceGMFear); v != 3 {
			t.Errorf("gm_fear = %d, want 3", v)
		}
		if v := ss.ResourceValue("unknown"); v != 0 {
			t.Errorf("unknown = %d, want 0", v)
		}
	})

	t.Run("ResourceCap", func(t *testing.T) {
		if v := ss.ResourceCap(ResourceGMFear); v != GMFearMax {
			t.Errorf("gm_fear cap = %d, want %d", v, GMFearMax)
		}
	})

	t.Run("ResourceNames", func(t *testing.T) {
		names := ss.ResourceNames()
		if len(names) != 1 || names[0] != ResourceGMFear {
			t.Errorf("names = %v, want [%s]", names, ResourceGMFear)
		}
	})

	t.Run("GMFear", func(t *testing.T) {
		if v := ss.GMFear(); v != 3 {
			t.Errorf("GMFear() = %d, want 3", v)
		}
	})
}

func TestStateFactory(t *testing.T) {
	factory := NewStateFactory()

	t.Run("NewCharacterState_PC", func(t *testing.T) {
		state, err := factory.NewCharacterState("camp-1", "char-1", 1) // CharacterKindPC = 1
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state.CampaignID() != "camp-1" {
			t.Errorf("CampaignID = %s, want camp-1", state.CampaignID())
		}
		if state.CharacterID() != "char-1" {
			t.Errorf("CharacterID = %s, want char-1", state.CharacterID())
		}
		// PC defaults
		if v := state.ResourceValue(ResourceHope); v != HopeDefault {
			t.Errorf("Hope = %d, want %d", v, HopeDefault)
		}
	})

	t.Run("NewSnapshotState", func(t *testing.T) {
		ss, err := factory.NewSnapshotState("camp-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ss.CampaignID() != "camp-1" {
			t.Errorf("CampaignID = %s, want camp-1", ss.CampaignID())
		}
		if v := ss.ResourceValue(ResourceGMFear); v != GMFearDefault {
			t.Errorf("GMFear = %d, want %d", v, GMFearDefault)
		}
	})
}

func TestCharacterState_SettersClamp(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		Hope:        4,
		HopeMax:     6,
		Stress:      2,
		StressMax:   6,
		Armor:       1,
		ArmorMax:    2,
	})

	state.SetHope(10)
	if state.Hope() != 6 {
		t.Fatalf("Hope = %d, want %d", state.Hope(), 6)
	}
	state.SetHopeMax(3)
	if state.HopeMax() != 3 {
		t.Fatalf("HopeMax = %d, want %d", state.HopeMax(), 3)
	}
	if state.Hope() != 3 {
		t.Fatalf("Hope after max clamp = %d, want %d", state.Hope(), 3)
	}

	state.SetStress(20)
	if state.Stress() != 6 {
		t.Fatalf("Stress = %d, want %d", state.Stress(), 6)
	}

	state.SetArmor(10)
	if state.Armor() != 2 {
		t.Fatalf("Armor = %d, want %d", state.Armor(), 2)
	}
}

func TestCharacterState_ArmorResource(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		Hope:        2,
		HopeMax:     6,
		Stress:      0,
		StressMax:   6,
		Armor:       1,
		ArmorMax:    2,
	})

	before, after, err := state.GainResource(ResourceArmor, 5)
	if err != nil {
		t.Fatalf("GainResource(armor): %v", err)
	}
	if before != 1 || after != 2 {
		t.Fatalf("GainResource(armor) = %d -> %d, want 1 -> 2", before, after)
	}

	before, after, err = state.SpendResource(ResourceArmor, 1)
	if err != nil {
		t.Fatalf("SpendResource(armor): %v", err)
	}
	if before != 2 || after != 1 {
		t.Fatalf("SpendResource(armor) = %d -> %d, want 2 -> 1", before, after)
	}

	_, _, err = state.SpendResource(ResourceArmor, 10)
	if err == nil {
		t.Fatal("expected insufficient armor error")
	}
}

func TestSnapshotState_SettersClamp(t *testing.T) {
	ss := NewSnapshotState(SnapshotStateConfig{CampaignID: "camp-1", GMFear: 1})

	ss.SetGMFear(20)
	if ss.GMFear() != GMFearMax {
		t.Fatalf("GMFear = %d, want %d", ss.GMFear(), GMFearMax)
	}
	ss.SetGMFear(-1)
	if ss.GMFear() != GMFearMin {
		t.Fatalf("GMFear = %d, want %d", ss.GMFear(), GMFearMin)
	}

	ss.SetShortRests(-5)
	if ss.ShortRests() != 0 {
		t.Fatalf("ShortRests = %d, want 0", ss.ShortRests())
	}
}

func TestClampMinGreaterThanMax(t *testing.T) {
	if got := clamp(5, 10, 1); got != 10 {
		t.Fatalf("clamp() = %d, want %d", got, 10)
	}
}

func TestGainStressEdgeCases(t *testing.T) {
	t.Run("ZeroAmount", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			HP: 6, HPMax: 6, Stress: 2, StressMax: 6,
			HopeMax: HopeMax,
		})
		result, err := state.GainStress(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.StressBefore != 2 || result.StressAfter != 2 {
			t.Fatalf("expected no change, got before=%d after=%d", result.StressBefore, result.StressAfter)
		}
		if result.HPBefore != 6 || result.HPAfter != 6 {
			t.Fatal("hp should not change")
		}
	})

	t.Run("NegativeAmount", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			HP: 6, HPMax: 6, Stress: 2, StressMax: 6,
			HopeMax: HopeMax,
		})
		result, err := state.GainStress(-1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.StressAfter != 2 {
			t.Fatal("stress should not change for negative amount")
		}
	})

	t.Run("AlreadyAtMax", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			HP: 6, HPMax: 6, Stress: 6, StressMax: 6,
			HopeMax: HopeMax,
		})
		result, err := state.GainStress(3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.StressAfter != 6 {
			t.Fatalf("expected stress capped at 6, got %d", result.StressAfter)
		}
		if result.Overflow != 3 {
			t.Fatalf("expected overflow=3, got %d", result.Overflow)
		}
		if result.HPAfter != 3 {
			t.Fatalf("expected hp=3 after overflow, got %d", result.HPAfter)
		}
	})

	t.Run("FillsExactly", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			HP: 6, HPMax: 6, Stress: 4, StressMax: 6,
			HopeMax: HopeMax,
		})
		result, err := state.GainStress(2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.LastStressMarked {
			t.Fatal("expected last_stress_marked")
		}
		if result.Overflow != 0 {
			t.Fatalf("expected no overflow, got %d", result.Overflow)
		}
		if result.StressAfter != 6 {
			t.Fatalf("expected stress=6, got %d", result.StressAfter)
		}
	})

	t.Run("OverflowWithHPDamage", func(t *testing.T) {
		state := NewCharacterState(CharacterStateConfig{
			HP: 6, HPMax: 6, Stress: 5, StressMax: 6,
			HopeMax: HopeMax,
		})
		result, err := state.GainStress(3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.LastStressMarked {
			t.Fatal("expected last_stress_marked")
		}
		if result.Overflow != 2 {
			t.Fatalf("expected overflow=2, got %d", result.Overflow)
		}
		if result.HPAfter != 4 {
			t.Fatalf("expected hp=4, got %d", result.HPAfter)
		}
	})
}

func TestResourceCapUnknown(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		HP: 6, HPMax: 6, HopeMax: HopeMax, StressMax: 6, ArmorMax: 2,
	})
	if v := state.ResourceCap("unknown"); v != 0 {
		t.Fatalf("expected 0 for unknown, got %d", v)
	}
}

func TestSnapshotResourceCapUnknown(t *testing.T) {
	ss := NewSnapshotState(SnapshotStateConfig{CampaignID: "camp-1"})
	if v := ss.ResourceCap("unknown"); v != 0 {
		t.Fatalf("expected 0 for unknown, got %d", v)
	}
}

func TestSnapshotSpendUnknownResource(t *testing.T) {
	ss := NewSnapshotState(SnapshotStateConfig{CampaignID: "camp-1", GMFear: 5})
	_, _, err := ss.SpendResource("unknown", 1)
	if err == nil {
		t.Fatal("expected error for unknown resource")
	}
}

func TestCharacterSpendStress(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		HP: 6, HPMax: 6, Stress: 3, StressMax: 6,
		HopeMax: HopeMax,
	})
	before, after, err := state.SpendResource(ResourceStress, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if before != 3 || after != 1 {
		t.Fatalf("expected 3->1, got %d->%d", before, after)
	}

	_, _, err = state.SpendResource(ResourceStress, 10)
	if err == nil {
		t.Fatal("expected error for insufficient stress")
	}
}

func TestCharacterSpendUnknownResource(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		HP: 6, HPMax: 6, HopeMax: HopeMax, StressMax: 6,
	})
	_, _, err := state.SpendResource("unknown", 1)
	if err == nil {
		t.Fatal("expected error for unknown resource")
	}
}

func TestStateFactory_NPC(t *testing.T) {
	factory := NewStateFactory()
	state, err := factory.NewCharacterState("camp-1", "npc-1", systems.CharacterKindNPC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NPC should have Hope=0 and StressMax=0
	if v := state.ResourceValue(ResourceHope); v != 0 {
		t.Errorf("NPC Hope = %d, want 0", v)
	}
	if v := state.ResourceCap(ResourceStress); v != 0 {
		t.Errorf("NPC Stress cap = %d, want 0", v)
	}
}

func TestNewCharacterStateDefaultLifeState(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		HopeMax:     HopeMax,
	})
	// LifeState should default to "alive"
	if state.lifeState != LifeStateAlive {
		t.Fatalf("expected life state alive, got %s", state.lifeState)
	}
}

func TestOutcomeApplier(t *testing.T) {
	applier := NewOutcomeApplier()
	changes, err := applier.ApplyOutcome(nil, systems.OutcomeContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changes != nil {
		t.Fatalf("expected nil changes, got %v", changes)
	}
}

func TestSystemMethods(t *testing.T) {
	sys := systems.DefaultRegistry.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	if sys == nil {
		t.Fatal("expected daggerheart system registered")
	}
	if sys.Name() != "Daggerheart" {
		t.Fatalf("Name() = %q, want Daggerheart", sys.Name())
	}
	if sys.StateFactory() == nil {
		t.Fatal("expected non-nil StateFactory")
	}
	if sys.OutcomeApplier() == nil {
		t.Fatal("expected non-nil OutcomeApplier")
	}
}
