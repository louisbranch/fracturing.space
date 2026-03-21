package daggerheart

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestNormalizedActiveBeastformPtr_ClampedFields(t *testing.T) {
	// Negative trait/evasion/damage bonuses clamped to 0.
	active := &CharacterActiveBeastformState{
		BeastformID:  "bf-1",
		TraitBonus:   -1,
		EvasionBonus: -1,
		DamageBonus:  -1,
		DamageDice:   []CharacterDamageDie{{Count: 2, Sides: 6}},
	}
	got := normalizedActiveBeastformPtr(active)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.TraitBonus != 0 {
		t.Fatalf("trait bonus = %d, want 0", got.TraitBonus)
	}
	if got.EvasionBonus != 0 {
		t.Fatalf("evasion bonus = %d, want 0", got.EvasionBonus)
	}
	if got.DamageBonus != 0 {
		t.Fatalf("damage bonus = %d, want 0", got.DamageBonus)
	}
}

func TestConditionState_UnmarshalJSON_StructuredForm(t *testing.T) {
	// Structured JSON (not legacy string).
	data := `{"id":"burn-1","class":"special","code":"burning","label":"Burning"}`
	var state ConditionState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		t.Fatalf("unmarshal structured: %v", err)
	}
	if state.ID != "burn-1" || state.Code != "burning" {
		t.Fatalf("state = %+v, want burn-1/burning", state)
	}

	// Legacy string.
	data = `"hidden"`
	state = ConditionState{}
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	if state.Standard != "hidden" {
		t.Fatalf("legacy state = %+v, want hidden", state)
	}

	// Invalid legacy string.
	data = `"unknown_condition"`
	state = ConditionState{}
	if err := json.Unmarshal([]byte(data), &state); err == nil {
		t.Fatal("expected error for invalid legacy condition")
	}

	// Invalid JSON.
	state = ConditionState{}
	if err := json.Unmarshal([]byte(`123`), &state); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCurrentBaseArmor(t *testing.T) {
	// No temporary armor.
	state := projectionstore.DaggerheartCharacterState{Armor: 3}
	if got := CurrentBaseArmor(state, 5); got != 3 {
		t.Fatalf("CurrentBaseArmor(3, 5) = %d, want 3", got)
	}
	// With temporary armor.
	state.TemporaryArmor = []projectionstore.DaggerheartTemporaryArmor{
		{Amount: 1, Duration: "short_rest"},
	}
	if got := CurrentBaseArmor(state, 5); got != 2 {
		t.Fatalf("CurrentBaseArmor(3-1, 5) = %d, want 2", got)
	}
	// Clamped to 0 when base is negative.
	state.Armor = 0
	if got := CurrentBaseArmor(state, 5); got != 0 {
		t.Fatalf("CurrentBaseArmor(0-1, 5) = %d, want 0", got)
	}
	// Clamped to armorMax.
	state.Armor = 10
	state.TemporaryArmor = nil
	if got := CurrentBaseArmor(state, 5); got != 5 {
		t.Fatalf("CurrentBaseArmor(10, 5) = %d, want 5", got)
	}
}

func TestNormalizeConditionStates_ErrorOnInvalid(t *testing.T) {
	// Invalid condition state in list.
	states := []ConditionState{
		{ID: "x", Class: "invalid"},
	}
	_, err := NormalizeConditionStates(states)
	if err == nil {
		t.Fatal("expected error for invalid condition class")
	}
}

func TestConditionStatesEqual_AllFieldsMismatch(t *testing.T) {
	a := []ConditionState{
		{ID: "hidden", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden"},
	}
	// Different source.
	b := []ConditionState{
		{ID: "hidden", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden", Source: "spell"},
	}
	if ConditionStatesEqual(a, b) {
		t.Fatal("different source should not be equal")
	}
	// Different source_id.
	c := []ConditionState{
		{ID: "hidden", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden", SourceID: "spell-1"},
	}
	if ConditionStatesEqual(a, c) {
		t.Fatal("different source_id should not be equal")
	}
}

func TestCompanionStateFromProjection(t *testing.T) {
	// nil returns nil.
	if got := companionStateFromProjection(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil converts.
	value := &projectionstore.DaggerheartCompanionState{
		Status:             CompanionStatusAway,
		ActiveExperienceID: "exp-1",
	}
	got := companionStateFromProjection(value)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Status != CompanionStatusAway || got.ActiveExperienceID != "exp-1" {
		t.Fatalf("got = %+v, want away/exp-1", got)
	}
}

func TestNormalizedClassStatePtr(t *testing.T) {
	// nil returns nil.
	if got := normalizedClassStatePtr(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil returns normalized copy.
	state := &CharacterClassState{AttackBonusUntilRest: -1, FocusTargetID: " char-1 "}
	got := normalizedClassStatePtr(state)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.AttackBonusUntilRest != 0 {
		t.Fatalf("attack bonus = %d, want 0", got.AttackBonusUntilRest)
	}
	if got.FocusTargetID != "char-1" {
		t.Fatalf("focus target = %q, want char-1", got.FocusTargetID)
	}
}

func TestHasClassStateFieldChange(t *testing.T) {
	// Both nil.
	if hasClassStateFieldChange(nil, nil) {
		t.Fatal("nil/nil should be no change")
	}
	// One nil.
	s := &CharacterClassState{}
	if !hasClassStateFieldChange(nil, s) {
		t.Fatal("nil/non-nil should be change")
	}
	if !hasClassStateFieldChange(s, nil) {
		t.Fatal("non-nil/nil should be change")
	}
	// Same values.
	a := &CharacterClassState{AttackBonusUntilRest: 2}
	b := &CharacterClassState{AttackBonusUntilRest: 2}
	if hasClassStateFieldChange(a, b) {
		t.Fatal("same values should be no change")
	}
	// Different values.
	c := &CharacterClassState{AttackBonusUntilRest: 3}
	if !hasClassStateFieldChange(a, c) {
		t.Fatal("different values should be change")
	}
}

func TestHasRestTakeMutation_AllBranches(t *testing.T) {
	// Fear change.
	if !hasRestTakeMutation(RestTakePayload{GMFearBefore: 1, GMFearAfter: 2}) {
		t.Fatal("fear change should be mutation")
	}
	// Short rests change.
	if !hasRestTakeMutation(RestTakePayload{ShortRestsBefore: 0, ShortRestsAfter: 1}) {
		t.Fatal("short rests change should be mutation")
	}
	// RefreshRest.
	if !hasRestTakeMutation(RestTakePayload{RefreshRest: true}) {
		t.Fatal("refresh rest should be mutation")
	}
	// RefreshLongRest.
	if !hasRestTakeMutation(RestTakePayload{RefreshLongRest: true}) {
		t.Fatal("refresh long rest should be mutation")
	}
	// Interrupted.
	if !hasRestTakeMutation(RestTakePayload{Interrupted: true}) {
		t.Fatal("interrupted should be mutation")
	}
	// CountdownUpdates.
	if !hasRestTakeMutation(RestTakePayload{CountdownUpdates: []CountdownUpdatePayload{{}}}) {
		t.Fatal("countdown updates should be mutation")
	}
	// DowntimeMoves.
	if !hasRestTakeMutation(RestTakePayload{DowntimeMoves: []DowntimeMoveAppliedPayload{{}}}) {
		t.Fatal("downtime moves should be mutation")
	}
	// Participants only.
	if !hasRestTakeMutation(RestTakePayload{Participants: []ids.CharacterID{"char-1"}}) {
		t.Fatal("participants should be mutation")
	}
	// No mutation (empty payload with zero values matching).
	if hasRestTakeMutation(RestTakePayload{}) {
		t.Fatal("empty payload should be no mutation")
	}
}

func TestNormalizeConditionStates_SortsByClassAndOrder(t *testing.T) {
	states := []ConditionState{
		{ID: "custom-1", Class: ConditionClassSpecial, Code: "burning"},
		{ID: "vulnerable", Class: ConditionClassStandard, Standard: "vulnerable", Code: "vulnerable"},
		{ID: "hidden", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden"},
	}
	got, err := NormalizeConditionStates(states)
	if err != nil {
		t.Fatalf("NormalizeConditionStates: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	// Sorted by class (special < standard alphabetically), then by order/code.
	if got[0].Code != "burning" {
		t.Fatalf("first = %q, want burning (special)", got[0].Code)
	}
	if got[1].Standard != "hidden" {
		t.Fatalf("second = %q, want hidden (standard)", got[1].Standard)
	}
	if got[2].Standard != "vulnerable" {
		t.Fatalf("third = %q, want vulnerable (standard)", got[2].Standard)
	}
}
