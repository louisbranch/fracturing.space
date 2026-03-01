package daggerheart

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	testcontracts "github.com/louisbranch/fracturing.space/internal/services/game/domain/internaltest/contracts"
)

func TestModuleEmittableEventTypes_MatchesDefinitions(t *testing.T) {
	module := NewModule()
	got := module.EmittableEventTypes()
	want := make([]event.Type, 0, len(daggerheartEventDefinitions))
	for _, def := range daggerheartEventDefinitions {
		want = append(want, def.Type)
	}
	if !testcontracts.EqualSlices(got, want) {
		t.Fatalf("EmittableEventTypes() = %v, want %v", got, want)
	}
}

func TestValidateCountdownCreatePayload_Branches(t *testing.T) {
	if err := validateCountdownCreatePayload(json.RawMessage(`{"countdown_id":"","name":"n","kind":"k","direction":"inc","max":1}`)); err == nil {
		t.Fatal("expected missing countdown_id error")
	}
	if err := validateCountdownCreatePayload(json.RawMessage(`{"countdown_id":"c1","name":"n","kind":"k","direction":"inc","max":0}`)); err == nil {
		t.Fatal("expected max-positive error")
	}
	if err := validateCountdownCreatePayload(json.RawMessage(`{"countdown_id":"c1","name":"n","kind":"k","direction":"inc","current":2,"max":1}`)); err == nil {
		t.Fatal("expected current range error")
	}
}

func TestValidateMultiTargetDamageApplyPayload_Branches(t *testing.T) {
	if err := validateMultiTargetDamageApplyPayload(json.RawMessage(`{"targets":[]}`)); err == nil {
		t.Fatal("expected targets required error")
	}
	if err := validateMultiTargetDamageApplyPayload(json.RawMessage(`{"targets":[{"character_id":"","hp_before":6,"hp_after":5}]}`)); err == nil {
		t.Fatal("expected target character_id required error")
	}
	if err := validateMultiTargetDamageApplyPayload(json.RawMessage(`{"targets":[{"character_id":"c1","hp_before":6,"hp_after":6}]}`)); err == nil {
		t.Fatal("expected mutation-required error")
	}
}

func TestHasConditionListMutation(t *testing.T) {
	if hasConditionListMutation([]string{" hidden "}, []string{"hidden"}) {
		t.Fatal("expected normalized equivalent lists to be no-op")
	}
	if !hasConditionListMutation([]string{"hidden"}, []string{"marked"}) {
		t.Fatal("expected different condition lists to be mutation")
	}
	if !hasConditionListMutation([]string{""}, []string{"hidden"}) {
		t.Fatal("expected invalid list to be treated as mutation")
	}
}

func TestHasRestCharacterStateMutationAndHasRestTakeMutation(t *testing.T) {
	one := 1
	two := 2
	if !hasRestCharacterStateMutation(RestCharacterStatePatch{HopeBefore: &one, HopeAfter: &two}) {
		t.Fatal("expected rest character mutation when hope changes")
	}
	if hasRestCharacterStateMutation(RestCharacterStatePatch{HopeBefore: &one, HopeAfter: &one}) {
		t.Fatal("expected no mutation when values are equal")
	}

	payload := RestTakePayload{
		CharacterStates: []RestCharacterStatePatch{{HopeBefore: &one, HopeAfter: &two}},
	}
	if !hasRestTakeMutation(payload) {
		t.Fatal("expected rest payload with character mutation to mutate")
	}
}

func TestValidateRestLongTermCountdownPayload(t *testing.T) {
	if err := validateRestLongTermCountdownPayload(CountdownUpdatePayload{}); err == nil {
		t.Fatal("expected missing countdown_id error")
	}
	if err := validateRestLongTermCountdownPayload(CountdownUpdatePayload{CountdownID: "c1"}); err == nil {
		t.Fatal("expected change-required error")
	}
	if err := validateRestLongTermCountdownPayload(CountdownUpdatePayload{CountdownID: "c1", Before: 1, After: 2}); err != nil {
		t.Fatalf("expected valid countdown payload, got %v", err)
	}
}

func TestIsTemporaryArmorDuration_HasStringFieldChange_AndAbs(t *testing.T) {
	if !isTemporaryArmorDuration("short_rest") || !isTemporaryArmorDuration("scene") {
		t.Fatal("expected valid temporary armor durations")
	}
	if isTemporaryArmorDuration("minute") {
		t.Fatal("expected invalid temporary armor duration to be rejected")
	}

	before := "a"
	after := "b"
	if !hasStringFieldChange(&before, &after) {
		t.Fatal("expected changed string fields to be mutation")
	}
	if hasStringFieldChange(&before, &before) {
		t.Fatal("expected equal string fields to be no-op")
	}
	if abs(-4) != 4 || abs(4) != 4 {
		t.Fatalf("abs() returned unexpected value")
	}
}
