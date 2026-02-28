package daggerheart

import "testing"

func TestEvaluateCreationProgress_NextStepAdvancesInOrder(t *testing.T) {
	progress := EvaluateCreationProgress(CreationProfile{})
	if progress.NextStep != CreationStepClassSubclass {
		t.Fatalf("next step = %d, want %d", progress.NextStep, CreationStepClassSubclass)
	}
	if progress.Ready {
		t.Fatal("ready = true, want false")
	}

	progress = EvaluateCreationProgress(CreationProfile{
		ClassID:        "class.guardian",
		SubclassID:     "subclass.stalwart",
		AncestryID:     "heritage.clank",
		CommunityID:    "heritage.farmer",
		TraitsAssigned: true,
		Traits: Traits{
			Agility:   2,
			Strength:  1,
			Finesse:   1,
			Instinct:  0,
			Presence:  0,
			Knowledge: -1,
		},
		DetailsRecorded:      true,
		Level:                1,
		HpMax:                6,
		StressMax:            6,
		Evasion:              10,
		StartingWeaponIDs:    []string{"weapon.longsword"},
		StartingArmorID:      "armor.gambeson-armor",
		StartingPotionItemID: StartingPotionMinorHealthID,
		Background:           "Guard captain",
		Experiences: []Experience{
			{Name: "Tactics", Modifier: 2},
		},
		DomainCardIDs: []string{"domain-card.ward"},
		Connections:   "Owes the blacksmith a favor",
	})
	if progress.NextStep != 0 {
		t.Fatalf("next step = %d, want 0", progress.NextStep)
	}
	if !progress.Ready {
		t.Fatal("ready = false, want true")
	}
}

func TestEvaluateCreationReadinessFromSystemProfile_RequiresDaggerheartProfile(t *testing.T) {
	ready, reason := EvaluateCreationReadinessFromSystemProfile(map[string]any{})
	if ready {
		t.Fatal("ready = true, want false")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestValidateCreationTraitDistribution(t *testing.T) {
	if err := ValidateCreationTraitDistribution(Traits{Agility: 2, Strength: 1, Finesse: 1, Instinct: 0, Presence: 0, Knowledge: -1}); err != nil {
		t.Fatalf("expected valid distribution, got %v", err)
	}
	if err := ValidateCreationTraitDistribution(Traits{Agility: 1, Strength: 1, Finesse: 1, Instinct: 0, Presence: 0, Knowledge: -1}); err == nil {
		t.Fatal("expected invalid distribution error")
	}
}

func TestIsValidStartingPotionItemID(t *testing.T) {
	if !IsValidStartingPotionItemID(StartingPotionMinorHealthID) {
		t.Fatalf("expected %q to be valid", StartingPotionMinorHealthID)
	}
	if !IsValidStartingPotionItemID(StartingPotionMinorStaminaID) {
		t.Fatalf("expected %q to be valid", StartingPotionMinorStaminaID)
	}
	if IsValidStartingPotionItemID("item.health-potion") {
		t.Fatal("expected non-starting potion to be invalid")
	}
}
