package mechanics

import (
	"testing"
)

// ── TierForLevel ────────────────────────────────────────────────────────

func TestTierForLevel(t *testing.T) {
	cases := []struct {
		level int
		tier  int
	}{
		{1, 1},
		{2, 2}, {3, 2}, {4, 2},
		{5, 3}, {6, 3}, {7, 3},
		{8, 4}, {9, 4}, {10, 4},
	}
	for _, tc := range cases {
		if got := TierForLevel(tc.level); got != tc.tier {
			t.Errorf("TierForLevel(%d) = %d, want %d", tc.level, got, tc.tier)
		}
	}
}

func TestTierForLevel_EdgeCases(t *testing.T) {
	// Below minimum clamps to tier 1.
	if got := TierForLevel(0); got != 1 {
		t.Errorf("TierForLevel(0) = %d, want 1", got)
	}
	// Above maximum stays at tier 4.
	if got := TierForLevel(11); got != 4 {
		t.Errorf("TierForLevel(11) = %d, want 4", got)
	}
}

// ── IsTierEntry ─────────────────────────────────────────────────────────

func TestIsTierEntry(t *testing.T) {
	entries := map[int]bool{
		1: false, 2: true, 3: false, 4: false,
		5: true, 6: false, 7: false,
		8: true, 9: false, 10: false,
	}
	for level, want := range entries {
		if got := IsTierEntry(level); got != want {
			t.Errorf("IsTierEntry(%d) = %v, want %v", level, got, want)
		}
	}
}

// ── AdvancementSlotCost ─────────────────────────────────────────────────

func TestAdvancementSlotCost(t *testing.T) {
	costOne := []AdvancementType{
		AdvTraitIncrease, AdvAddHPSlots, AdvAddStressSlots,
		AdvIncreaseExperience, AdvDomainCard, AdvIncreaseEvasion,
		AdvUpgradedSubclass,
	}
	for _, adv := range costOne {
		if got := AdvancementSlotCost(adv); got != 1 {
			t.Errorf("AdvancementSlotCost(%s) = %d, want 1", adv, got)
		}
	}
	costTwo := []AdvancementType{AdvIncreaseProficiency, AdvMulticlass}
	for _, adv := range costTwo {
		if got := AdvancementSlotCost(adv); got != 2 {
			t.Errorf("AdvancementSlotCost(%s) = %d, want 2", adv, got)
		}
	}
}

// ── IsValidTrait ────────────────────────────────────────────────────────

func TestIsValidTrait(t *testing.T) {
	for _, name := range AllTraitNames() {
		if !IsValidTrait(string(name)) {
			t.Errorf("IsValidTrait(%q) = false, want true", name)
		}
	}
	if IsValidTrait("charisma") {
		t.Error("IsValidTrait(charisma) = true, want false")
	}
}

// ── ValidateLevelUp ─────────────────────────────────────────────────────

func TestValidateLevelUp_BasicLevelUp(t *testing.T) {
	req := LevelUpRequest{
		CharacterID: "char-1",
		LevelBefore: 1,
		LevelAfter:  2,
		Advancements: []Advancement{
			{Type: AdvTraitIncrease, Trait: "agility"},
			{Type: AdvAddHPSlots},
		},
	}
	result, err := ValidateLevelUp(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Tier != 2 {
		t.Errorf("Tier = %d, want 2", result.Tier)
	}
	if result.PreviousTier != 1 {
		t.Errorf("PreviousTier = %d, want 1", result.PreviousTier)
	}
	if !result.IsTierEntry {
		t.Error("IsTierEntry = false, want true (level 2 enters T2)")
	}
	if result.SlotsConsumed != 2 {
		t.Errorf("SlotsConsumed = %d, want 2", result.SlotsConsumed)
	}
	if result.ThresholdDelta != 1 {
		t.Errorf("ThresholdDelta = %d, want 1", result.ThresholdDelta)
	}
}

func TestValidateLevelUp_TierEntryLevel5_ClearsMarks(t *testing.T) {
	req := LevelUpRequest{
		CharacterID:  "char-1",
		LevelBefore:  4,
		LevelAfter:   5,
		MarkedTraits: []string{"agility", "strength"},
		Advancements: []Advancement{
			// Can re-increase agility because marks clear at tier entry.
			{Type: AdvTraitIncrease, Trait: "agility"},
			{Type: AdvAddStressSlots},
		},
	}
	result, err := ValidateLevelUp(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ClearMarks {
		t.Error("ClearMarks = false, want true (entering T3)")
	}
	if !result.IsTierEntry {
		t.Error("IsTierEntry = false, want true")
	}
	// After clearing and re-marking agility, only agility should be marked.
	if len(result.MarkedTraits) != 1 || result.MarkedTraits[0] != "agility" {
		t.Errorf("MarkedTraits = %v, want [agility]", result.MarkedTraits)
	}
}

func TestValidateLevelUp_TierEntryLevel8_ClearsMarks(t *testing.T) {
	req := LevelUpRequest{
		CharacterID:  "char-1",
		LevelBefore:  7,
		LevelAfter:   8,
		MarkedTraits: []string{"finesse"},
		Advancements: []Advancement{
			{Type: AdvTraitIncrease, Trait: "finesse"},
			{Type: AdvIncreaseEvasion},
		},
	}
	result, err := ValidateLevelUp(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ClearMarks {
		t.Error("ClearMarks = false, want true (entering T4)")
	}
}

func TestValidateLevelUp_Level2TierEntry_DoesNotClearMarks(t *testing.T) {
	// Level 2 is a tier entry but NOT a mark-clearing level (only 5 and 8 clear).
	req := LevelUpRequest{
		CharacterID: "char-1",
		LevelBefore: 1,
		LevelAfter:  2,
		Advancements: []Advancement{
			{Type: AdvAddHPSlots},
			{Type: AdvAddStressSlots},
		},
	}
	result, err := ValidateLevelUp(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClearMarks {
		t.Error("ClearMarks = true, want false (level 2 doesn't clear marks)")
	}
}

func TestValidateLevelUp_NonTierEntry(t *testing.T) {
	req := LevelUpRequest{
		CharacterID: "char-1",
		LevelBefore: 2,
		LevelAfter:  3,
		Advancements: []Advancement{
			{Type: AdvTraitIncrease, Trait: "presence"},
			{Type: AdvIncreaseExperience},
		},
	}
	result, err := ValidateLevelUp(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsTierEntry {
		t.Error("IsTierEntry = true, want false")
	}
	if result.ClearMarks {
		t.Error("ClearMarks = true, want false")
	}
}

func TestValidateLevelUp_ProficiencyCosts2Slots(t *testing.T) {
	req := LevelUpRequest{
		CharacterID: "char-1",
		LevelBefore: 3,
		LevelAfter:  4,
		Advancements: []Advancement{
			{Type: AdvIncreaseProficiency},
		},
	}
	result, err := ValidateLevelUp(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SlotsConsumed != 2 {
		t.Errorf("SlotsConsumed = %d, want 2", result.SlotsConsumed)
	}
}

// ── Validation error cases ──────────────────────────────────────────────

func TestValidateLevelUp_RejectsLevelBelow1(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore:  0,
		LevelAfter:   1,
		Advancements: []Advancement{{Type: AdvAddHPSlots}, {Type: AdvAddStressSlots}},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for level_before=0")
	}
}

func TestValidateLevelUp_RejectsLevelAt10(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore:  10,
		LevelAfter:   11,
		Advancements: []Advancement{{Type: AdvAddHPSlots}, {Type: AdvAddStressSlots}},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for level_before=10")
	}
}

func TestValidateLevelUp_RejectsNonConsecutiveLevel(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore:  2,
		LevelAfter:   4,
		Advancements: []Advancement{{Type: AdvAddHPSlots}, {Type: AdvAddStressSlots}},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for non-consecutive level")
	}
}

func TestValidateLevelUp_RejectsWrongSlotCount(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore: 1,
		LevelAfter:  2,
		Advancements: []Advancement{
			{Type: AdvAddHPSlots},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for 1 slot consumed instead of 2")
	}
}

func TestValidateLevelUp_RejectsOverBudget(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore: 1,
		LevelAfter:  2,
		Advancements: []Advancement{
			{Type: AdvAddHPSlots},
			{Type: AdvAddStressSlots},
			{Type: AdvIncreaseEvasion},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for 3 slots consumed")
	}
}

func TestValidateLevelUp_RejectsMarkedTrait(t *testing.T) {
	req := LevelUpRequest{
		CharacterID:  "char-1",
		LevelBefore:  2,
		LevelAfter:   3,
		MarkedTraits: []string{"agility"},
		Advancements: []Advancement{
			{Type: AdvTraitIncrease, Trait: "agility"},
			{Type: AdvAddHPSlots},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for increasing a marked trait")
	}
}

func TestValidateLevelUp_RejectsTraitIncreaseMissingTrait(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore: 1,
		LevelAfter:  2,
		Advancements: []Advancement{
			{Type: AdvTraitIncrease},
			{Type: AdvAddHPSlots},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for trait_increase without trait")
	}
}

func TestValidateLevelUp_RejectsUnknownTrait(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore: 1,
		LevelAfter:  2,
		Advancements: []Advancement{
			{Type: AdvTraitIncrease, Trait: "charisma"},
			{Type: AdvAddHPSlots},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for unknown trait")
	}
}

func TestValidateLevelUp_RejectsDuplicateTraitIncrease(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore: 1,
		LevelAfter:  2,
		Advancements: []Advancement{
			{Type: AdvTraitIncrease, Trait: "agility"},
			{Type: AdvTraitIncrease, Trait: "agility"},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for duplicate trait increase in same level-up")
	}
}

func TestValidateLevelUp_RejectsMulticlassBeforeLevel5(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore: 3,
		LevelAfter:  4,
		Advancements: []Advancement{
			{Type: AdvMulticlass, Multiclass: &MulticlassChoice{
				SecondaryClassID:    "warrior",
				SecondarySubclassID: "slayer",
				SpellcastTrait:      "presence",
				DomainID:            "blade",
			}},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for multiclass before level 5")
	}
}

func TestValidateLevelUp_AcceptsMulticlassAtLevel5(t *testing.T) {
	req := LevelUpRequest{
		CharacterID: "char-1",
		LevelBefore: 4,
		LevelAfter:  5,
		Advancements: []Advancement{
			{Type: AdvMulticlass, Multiclass: &MulticlassChoice{
				SecondaryClassID:    "warrior",
				SecondarySubclassID: "slayer",
				SpellcastTrait:      "presence",
				DomainID:            "blade",
			}},
		},
	}
	result, err := ValidateLevelUp(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SlotsConsumed != 2 {
		t.Errorf("SlotsConsumed = %d, want 2", result.SlotsConsumed)
	}
}

func TestValidateLevelUp_RejectsMulticlassMissingChoices(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore: 4,
		LevelAfter:  5,
		Advancements: []Advancement{
			{Type: AdvMulticlass},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for multiclass without choices")
	}
}

func TestValidateLevelUp_RejectsMulticlassMissingFields(t *testing.T) {
	cases := []struct {
		name string
		mc   MulticlassChoice
	}{
		{"missing class", MulticlassChoice{SecondarySubclassID: "s", DomainID: "d"}},
		{"missing subclass", MulticlassChoice{SecondaryClassID: "c", DomainID: "d"}},
		{"missing domain", MulticlassChoice{SecondaryClassID: "c", SecondarySubclassID: "s"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := LevelUpRequest{
				LevelBefore: 4,
				LevelAfter:  5,
				Advancements: []Advancement{
					{Type: AdvMulticlass, Multiclass: &tc.mc},
				},
			}
			_, err := ValidateLevelUp(req)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateLevelUp_RejectsDomainCardLevelExceedsCharacterLevel(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore: 2,
		LevelAfter:  3,
		Advancements: []Advancement{
			{Type: AdvDomainCard, DomainCardID: "card-1", DomainCardLevel: 5},
			{Type: AdvAddHPSlots},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for domain card level exceeding character level")
	}
}

func TestValidateLevelUp_RejectsUnknownAdvancementType(t *testing.T) {
	req := LevelUpRequest{
		LevelBefore: 1,
		LevelAfter:  2,
		Advancements: []Advancement{
			{Type: "unknown"},
			{Type: AdvAddHPSlots},
		},
	}
	_, err := ValidateLevelUp(req)
	if err == nil {
		t.Fatal("expected error for unknown advancement type")
	}
}
