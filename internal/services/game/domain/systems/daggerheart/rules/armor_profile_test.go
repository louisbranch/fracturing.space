package rules

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestApplyArmorProfileEffects_AppliesPassiveModifiers(t *testing.T) {
	armor := &contentstore.DaggerheartArmor{
		ID:                  "armor.channeling-armor",
		BaseMajorThreshold:  13,
		BaseSevereThreshold: 36,
		ArmorScore:          5,
		Rules: contentstore.DaggerheartArmorRules{
			EvasionDelta:       -1,
			AllTraitsDelta:     -1,
			SpellcastRollBonus: 1,
		},
	}

	profile := ApplyArmorProfileEffects(4, ArmorPassiveBase{
		Evasion: 12,
		Traits:  testArmorProfileTraits(2, 1, 1, 0, 0, -1),
	}, armor)

	if profile.EquippedArmorID != armor.ID {
		t.Fatalf("equipped armor = %q, want %q", profile.EquippedArmorID, armor.ID)
	}
	if profile.Evasion != 11 {
		t.Fatalf("evasion = %d, want 11", profile.Evasion)
	}
	if profile.Agility != 1 || profile.Strength != 0 || profile.Presence != -1 || profile.Knowledge != -2 {
		t.Fatalf("traits = %+v, want agility/strength/presence/knowledge adjusted", profile)
	}
	if profile.ArmorScore != 5 || profile.ArmorMax != 5 {
		t.Fatalf("armor score/max = %d/%d, want 5/5", profile.ArmorScore, profile.ArmorMax)
	}
	if profile.SpellcastRollBonus != 1 {
		t.Fatalf("spellcast roll bonus = %d, want 1", profile.SpellcastRollBonus)
	}
}

func TestEffectiveArmorRules_DefaultsAndOverrides(t *testing.T) {
	t.Run("nil armor uses safe defaults", func(t *testing.T) {
		rules := EffectiveArmorRules(nil)
		if rules.AutomationStatus != contentstore.DaggerheartArmorAutomationStatusSupported {
			t.Fatalf("automation_status = %q, want supported", rules.AutomationStatus)
		}
		if rules.MitigationMode != contentstore.DaggerheartArmorMitigationModeAny {
			t.Fatalf("mitigation_mode = %q, want any", rules.MitigationMode)
		}
		if rules.SeverityReductionSteps != 1 {
			t.Fatalf("severity_reduction_steps = %d, want 1", rules.SeverityReductionSteps)
		}
	})

	t.Run("blank rule fields keep defaults while explicit overrides apply", func(t *testing.T) {
		rules := EffectiveArmorRules(&contentstore.DaggerheartArmor{
			Rules: contentstore.DaggerheartArmorRules{
				AutomationStatus:                "",
				MitigationMode:                  "",
				SeverityReductionSteps:          0,
				EvasionDelta:                    -1,
				AgilityDelta:                    1,
				PresenceDelta:                   -2,
				SpellcastRollBonus:              2,
				AllTraitsDelta:                  -1,
				StressOnMark:                    true,
				ThresholdBonusWhenArmorDepleted: 3,
				WardedMagicReduction:            true,
				SilentMovementBonus:             2,
			},
		})
		if rules.AutomationStatus != contentstore.DaggerheartArmorAutomationStatusSupported {
			t.Fatalf("automation_status = %q, want supported default", rules.AutomationStatus)
		}
		if rules.MitigationMode != contentstore.DaggerheartArmorMitigationModeAny {
			t.Fatalf("mitigation_mode = %q, want any default", rules.MitigationMode)
		}
		if rules.SeverityReductionSteps != 1 {
			t.Fatalf("severity_reduction_steps = %d, want 1 default", rules.SeverityReductionSteps)
		}
		if rules.SpellcastRollBonus != 2 || !rules.StressOnMark || !rules.WardedMagicReduction || rules.SilentMovementBonus != 2 {
			t.Fatalf("explicit overrides were not preserved: %+v", rules)
		}
	})
}

func TestRemoveArmorPassiveEffects_ReconstructsBaseTraits(t *testing.T) {
	currentArmor := &contentstore.DaggerheartArmor{
		ID: "armor.savior-chainmail",
		Rules: contentstore.DaggerheartArmorRules{
			EvasionDelta:   -1,
			AllTraitsDelta: -1,
			PresenceDelta:  1,
		},
	}

	base := RemoveArmorPassiveEffects(projectionstore.DaggerheartCharacterProfile{
		Evasion:   9,
		Agility:   1,
		Strength:  0,
		Finesse:   0,
		Instinct:  -1,
		Presence:  0,
		Knowledge: -2,
	}, currentArmor)

	if base.Evasion != 10 {
		t.Fatalf("base evasion = %d, want 10", base.Evasion)
	}
	if base.Traits.Agility != 2 || base.Traits.Strength != 1 || base.Traits.Presence != 0 || base.Traits.Knowledge != -1 {
		t.Fatalf("base traits = %+v, want original armorless values", base.Traits)
	}
}

func TestRemoveArmorPassiveEffects_WithoutArmorReturnsStoredValues(t *testing.T) {
	base := RemoveArmorPassiveEffects(projectionstore.DaggerheartCharacterProfile{
		Evasion:   11,
		Agility:   2,
		Strength:  1,
		Finesse:   0,
		Instinct:  -1,
		Presence:  3,
		Knowledge: -2,
	}, nil)

	if base.Evasion != 11 {
		t.Fatalf("base evasion = %d, want 11", base.Evasion)
	}
	if base.Traits != testArmorProfileTraits(2, 1, 0, -1, 3, -2) {
		t.Fatalf("base traits = %+v, want unchanged", base.Traits)
	}
}

func TestApplyArmorProfileEffects_UnarmoredUsesDerivedThresholds(t *testing.T) {
	base := ArmorPassiveBase{
		Evasion: 10,
		Traits:  testArmorProfileTraits(2, 1, 0, 0, -1, -2),
	}

	profile := ApplyArmorProfileEffects(3, base, nil)
	wantMajor, wantSevere := daggerheartprofile.UnarmoredThresholds(3)
	if profile.MajorThreshold != wantMajor || profile.SevereThreshold != wantSevere {
		t.Fatalf("thresholds = %d/%d, want %d/%d", profile.MajorThreshold, profile.SevereThreshold, wantMajor, wantSevere)
	}
	if profile.EquippedArmorID != "" || profile.ArmorScore != 0 || profile.ArmorMax != 0 {
		t.Fatalf("unarmored profile should not carry armor data: %+v", profile)
	}
}

func TestRemapArmorCurrent_PreservesMarkedSlotsAcrossSwap(t *testing.T) {
	got := RemapArmorCurrent(projectionstore.DaggerheartCharacterState{
		Armor: 3,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Amount: 1},
		},
	}, 5, 4)

	if got != 2 {
		t.Fatalf("remapped armor = %d, want 2", got)
	}
}

func TestRemapArmorCurrent_ClampsBaseArmorAndTemporaryBuckets(t *testing.T) {
	t.Run("temporary armor can exceed current total without going negative", func(t *testing.T) {
		got := RemapArmorCurrent(projectionstore.DaggerheartCharacterState{
			Armor: 1,
			TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
				{Amount: 2},
			},
		}, 3, 2)
		if got != 2 {
			t.Fatalf("remapped armor = %d, want 2", got)
		}
	})

	t.Run("current armor above old max clamps before preserving marks", func(t *testing.T) {
		got := RemapArmorCurrent(projectionstore.DaggerheartCharacterState{
			Armor: 8,
		}, 5, 4)
		if got != 4 {
			t.Fatalf("remapped armor = %d, want 4", got)
		}
	})

	t.Run("new armor cannot fall below zero", func(t *testing.T) {
		got := RemapArmorCurrent(projectionstore.DaggerheartCharacterState{
			Armor: 0,
		}, 4, 1)
		if got != 0 {
			t.Fatalf("remapped armor = %d, want 0", got)
		}
	})
}

func TestBaseArmorHelpers_PreserveTemporaryArmor(t *testing.T) {
	state := projectionstore.DaggerheartCharacterState{
		Armor: 4,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Amount: 1},
			{Amount: -1},
			{Amount: 2},
		},
	}

	if got := CurrentBaseArmor(state, 3); got != 1 {
		t.Fatalf("current base armor = %d, want 1", got)
	}
	if !IsLastBaseArmorSlot(state, 3) {
		t.Fatal("expected last base armor slot")
	}
	beforeBase, afterBase, ok := SpendBaseArmorSlot(state, 3)
	if !ok || beforeBase != 1 || afterBase != 0 {
		t.Fatalf("spend base armor = %d -> %d ok=%v, want 1 -> 0 true", beforeBase, afterBase, ok)
	}
	beforeTotal, afterTotal, ok := ArmorTotalAfterBaseSpend(state, 3)
	if !ok || beforeTotal != 4 || afterTotal != 3 {
		t.Fatalf("armor total after spend = %d -> %d ok=%v, want 4 -> 3 true", beforeTotal, afterTotal, ok)
	}
}

func TestBaseArmorHelpers_RejectWhenNoBaseArmorRemains(t *testing.T) {
	state := projectionstore.DaggerheartCharacterState{
		Armor: 2,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Amount: 3},
		},
	}

	if got := CurrentBaseArmor(state, 5); got != 0 {
		t.Fatalf("current base armor = %d, want 0", got)
	}
	if IsLastBaseArmorSlot(state, 5) {
		t.Fatal("did not expect last base armor slot when no base armor remains")
	}
	beforeBase, afterBase, ok := SpendBaseArmorSlot(state, 5)
	if ok || beforeBase != 0 || afterBase != 0 {
		t.Fatalf("spend base armor = %d -> %d ok=%v, want 0 -> 0 false", beforeBase, afterBase, ok)
	}
	beforeTotal, afterTotal, ok := ArmorTotalAfterBaseSpend(state, 5)
	if ok || beforeTotal != 2 || afterTotal != 2 {
		t.Fatalf("armor total after rejected spend = %d -> %d ok=%v, want 2 -> 2 false", beforeTotal, afterTotal, ok)
	}
}

func testArmorProfileTraits(agility, strength, finesse, instinct, presence, knowledge int) daggerheartprofile.Traits {
	return daggerheartprofile.Traits{
		Agility:   agility,
		Strength:  strength,
		Finesse:   finesse,
		Instinct:  instinct,
		Presence:  presence,
		Knowledge: knowledge,
	}
}
