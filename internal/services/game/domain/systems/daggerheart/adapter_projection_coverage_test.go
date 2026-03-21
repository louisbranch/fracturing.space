package daggerheart

import (
	"testing"

	daggerheartadapter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/adapter"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestSubclassStateToProjection(t *testing.T) {
	// nil returns nil.
	if got := daggerheartadapter.SubclassStateToProjection(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil converts.
	state := &daggerheartstate.CharacterSubclassState{
		BattleRitualUsedThisLongRest: true,
		ElementalChannel:             "fire",
		NemesisTargetID:              "adv-1",
	}
	got := daggerheartadapter.SubclassStateToProjection(state)
	if got == nil {
		t.Fatal("expected non-nil projection")
	}
	if !got.BattleRitualUsedThisLongRest {
		t.Fatal("BattleRitualUsedThisLongRest should be true")
	}
	if got.ElementalChannel != "fire" {
		t.Fatalf("ElementalChannel = %q, want fire", got.ElementalChannel)
	}
	if got.NemesisTargetID != "adv-1" {
		t.Fatalf("NemesisTargetID = %q, want adv-1", got.NemesisTargetID)
	}
}

func TestSubclassStateFromProjection(t *testing.T) {
	// nil returns nil.
	if got := daggerheartadapter.SubclassStateFromProjection(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil round-trips correctly.
	value := &projectionstore.DaggerheartSubclassState{
		BattleRitualUsedThisLongRest: true,
		ElementalChannel:             "fire",
		NemesisTargetID:              "adv-1",
	}
	got := daggerheartadapter.SubclassStateFromProjection(value)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if !got.BattleRitualUsedThisLongRest {
		t.Fatal("BattleRitualUsedThisLongRest should be true")
	}
	if got.ElementalChannel != "fire" {
		t.Fatalf("ElementalChannel = %q, want fire", got.ElementalChannel)
	}
}

func TestConditionStatesToProjection(t *testing.T) {
	// nil returns nil.
	if got := daggerheartadapter.ConditionStatesToProjection(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Converts with clear triggers.
	states := []rules.ConditionState{
		{
			ID:            "hidden",
			Class:         rules.ConditionClassStandard,
			Standard:      "hidden",
			Code:          "hidden",
			Label:         "Hidden",
			Source:        "spell",
			SourceID:      "spell-1",
			ClearTriggers: []rules.ConditionClearTrigger{rules.ConditionClearTriggerShortRest, rules.ConditionClearTriggerDamageTaken},
		},
		{
			ID:    "burning",
			Class: rules.ConditionClassSpecial,
			Code:  "burning",
			Label: "Burning",
		},
	}
	got := daggerheartadapter.ConditionStatesToProjection(states)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "hidden" || got[0].Source != "spell" {
		t.Fatalf("first = %+v, want hidden with source", got[0])
	}
	if len(got[0].ClearTriggers) != 2 {
		t.Fatalf("clear triggers len = %d, want 2", len(got[0].ClearTriggers))
	}
	if got[1].ID != "burning" || len(got[1].ClearTriggers) != 0 {
		t.Fatalf("second = %+v, want burning without triggers", got[1])
	}
}

func TestToProjectionAdversaryFeatureStates(t *testing.T) {
	in := []rules.AdversaryFeatureState{
		{FeatureID: " f1 ", Status: " active ", FocusedTargetID: " char-1 "},
		{FeatureID: "f2", Status: "used"},
	}
	got := daggerheartadapter.ToProjectionAdversaryFeatureStates(in)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].FeatureID != "f1" || got[0].Status != "active" || got[0].FocusedTargetID != "char-1" {
		t.Fatalf("first = %+v, want trimmed values", got[0])
	}
}

func TestToProjectionAdversaryPendingExperience(t *testing.T) {
	// nil returns nil.
	if got := daggerheartadapter.ToProjectionAdversaryPendingExperience(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil converts.
	in := &rules.AdversaryPendingExperience{Name: " xp ", Modifier: 10}
	got := daggerheartadapter.ToProjectionAdversaryPendingExperience(in)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Name != "xp" || got.Modifier != 10 {
		t.Fatalf("got = %+v, want trimmed values", got)
	}
}

func TestClassStateToProjection(t *testing.T) {
	// nil returns nil.
	if got := daggerheartadapter.ClassStateToProjection(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil converts correctly.
	state := &daggerheartstate.CharacterClassState{
		AttackBonusUntilRest: 2,
		FocusTargetID:        "char-1",
		RallyDice:            []int{6, 8},
		Unstoppable: daggerheartstate.CharacterUnstoppableState{
			Active:       true,
			CurrentValue: 3,
			DieSides:     6,
		},
	}
	got := daggerheartadapter.ClassStateToProjection(state)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.AttackBonusUntilRest != 2 {
		t.Fatalf("attack bonus = %d, want 2", got.AttackBonusUntilRest)
	}
	if len(got.RallyDice) != 2 {
		t.Fatalf("rally dice len = %d, want 2", len(got.RallyDice))
	}
	if !got.Unstoppable.Active {
		t.Fatal("unstoppable should be active")
	}
}

func TestCompanionStateToProjection(t *testing.T) {
	// nil returns nil.
	if got := daggerheartadapter.CompanionStateToProjection(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil converts.
	state := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusAway, ActiveExperienceID: "exp-1"}
	got := daggerheartadapter.CompanionStateToProjection(state)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Status != daggerheartstate.CompanionStatusAway || got.ActiveExperienceID != "exp-1" {
		t.Fatalf("got = %+v, want away/exp-1", got)
	}
}

func TestActiveBeastformToProjection(t *testing.T) {
	// nil returns nil.
	if got := daggerheartadapter.ActiveBeastformToProjection(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil converts with damage dice.
	state := &daggerheartstate.CharacterActiveBeastformState{
		BeastformID: "bf-1",
		BaseTrait:   "strength",
		AttackTrait: "strength",
		DamageDice:  []daggerheartstate.CharacterDamageDie{{Count: 2, Sides: 6}},
		DamageType:  "physical",
	}
	got := daggerheartadapter.ActiveBeastformToProjection(state)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.BeastformID != "bf-1" {
		t.Fatalf("beastform id = %q, want bf-1", got.BeastformID)
	}
	if len(got.DamageDice) != 1 || got.DamageDice[0].Count != 2 {
		t.Fatalf("damage dice = %v, want [{2 6}]", got.DamageDice)
	}
}

func TestClassStateFromProjection(t *testing.T) {
	// Without beastform.
	value := projectionstore.DaggerheartClassState{
		AttackBonusUntilRest: 2,
		RallyDice:            []int{6},
	}
	got := daggerheartadapter.ClassStateFromProjection(value)
	if got.AttackBonusUntilRest != 2 {
		t.Fatalf("attack bonus = %d, want 2", got.AttackBonusUntilRest)
	}
	if got.ActiveBeastform != nil {
		t.Fatal("expected nil beastform without projection beastform")
	}

	// With beastform.
	value.ActiveBeastform = &projectionstore.DaggerheartActiveBeastformState{
		BeastformID: "bf-1",
		DamageDice:  []projectionstore.DaggerheartDamageDie{{Count: 1, Sides: 8}},
	}
	got = daggerheartadapter.ClassStateFromProjection(value)
	if got.ActiveBeastform == nil || got.ActiveBeastform.BeastformID != "bf-1" {
		t.Fatalf("beastform = %+v, want bf-1", got.ActiveBeastform)
	}
}
