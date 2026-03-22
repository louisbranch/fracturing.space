package daggerheart

import (
	"encoding/json"
	"testing"

	daggerheartvalidator "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/validator"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
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
	if err := daggerheartvalidator.ValidateSceneCountdownCreatePayload(json.RawMessage(`{"countdown_id":"","name":"n","tone":"progress","advancement_policy":"manual","starting_value":1,"remaining_value":1,"loop_behavior":"none","status":"active"}`)); err == nil {
		t.Fatal("expected missing countdown_id error")
	}
	if err := daggerheartvalidator.ValidateSceneCountdownCreatePayload(json.RawMessage(`{"countdown_id":"c1","name":"n","tone":"progress","advancement_policy":"manual","starting_value":0,"remaining_value":0,"loop_behavior":"none","status":"active"}`)); err == nil {
		t.Fatal("expected starting_value-positive error")
	}
	if err := daggerheartvalidator.ValidateSceneCountdownCreatePayload(json.RawMessage(`{"countdown_id":"c1","name":"n","tone":"progress","advancement_policy":"manual","starting_value":1,"remaining_value":2,"loop_behavior":"none","status":"active"}`)); err == nil {
		t.Fatal("expected remaining_value range error")
	}
}

func TestValidateMultiTargetDamageApplyPayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateMultiTargetDamageApplyPayload(json.RawMessage(`{"targets":[]}`)); err == nil {
		t.Fatal("expected targets required error")
	}
	if err := daggerheartvalidator.ValidateMultiTargetDamageApplyPayload(json.RawMessage(`{"targets":[{"character_id":"","hp_before":6,"hp_after":5}]}`)); err == nil {
		t.Fatal("expected target character_id required error")
	}
	if err := daggerheartvalidator.ValidateMultiTargetDamageApplyPayload(json.RawMessage(`{"targets":[{"character_id":"c1","hp_before":6,"hp_after":6}]}`)); err == nil {
		t.Fatal("expected mutation-required error")
	}
}

func TestHasConditionListMutation(t *testing.T) {
	if daggerheartvalidator.HasConditionListMutation([]string{" hidden "}, []string{"hidden"}) {
		t.Fatal("expected normalized equivalent lists to be no-op")
	}
	if !daggerheartvalidator.HasConditionListMutation([]string{"hidden"}, []string{"marked"}) {
		t.Fatal("expected different condition lists to be mutation")
	}
	if !daggerheartvalidator.HasConditionListMutation([]string{""}, []string{"hidden"}) {
		t.Fatal("expected invalid list to be treated as mutation")
	}
}

func TestHasRestTakeMutation(t *testing.T) {
	payload := daggerheartpayload.RestTakePayload{
		Participants: []ids.CharacterID{"char-1"},
		Interrupted:  true,
	}
	if !daggerheartvalidator.HasRestTakeMutation(payload) {
		t.Fatal("expected interrupted rest payload with participants to mutate")
	}
}

func TestValidateRestLongTermCountdownPayload(t *testing.T) {
	if err := daggerheartvalidator.ValidateRestLongTermCountdownPayload(daggerheartpayload.CountdownUpdatePayload{}); err == nil {
		t.Fatal("expected missing countdown_id error")
	}
	if err := daggerheartvalidator.ValidateRestLongTermCountdownPayload(daggerheartpayload.CountdownUpdatePayload{CountdownID: "c1"}); err == nil {
		t.Fatal("expected change-required error")
	}
	if err := daggerheartvalidator.ValidateRestLongTermCountdownPayload(daggerheartpayload.CountdownUpdatePayload{CountdownID: "c1", BeforeRemaining: 1, AfterRemaining: 2, AdvancedBy: 1}); err != nil {
		t.Fatalf("expected valid countdown payload, got %v", err)
	}
}

func TestIsTemporaryArmorDuration_HasStringFieldChange_AndAbs(t *testing.T) {
	if !daggerheartvalidator.IsTemporaryArmorDuration("short_rest") || !daggerheartvalidator.IsTemporaryArmorDuration("scene") {
		t.Fatal("expected valid temporary armor durations")
	}
	if daggerheartvalidator.IsTemporaryArmorDuration("minute") {
		t.Fatal("expected invalid temporary armor duration to be rejected")
	}

	before := "a"
	after := "b"
	if !daggerheartvalidator.HasStringFieldChange(&before, &after) {
		t.Fatal("expected changed string fields to be mutation")
	}
	if daggerheartvalidator.HasStringFieldChange(&before, &before) {
		t.Fatal("expected equal string fields to be no-op")
	}
	if daggerheartvalidator.Abs(-4) != 4 || daggerheartvalidator.Abs(4) != 4 {
		t.Fatalf("abs() returned unexpected value")
	}
}
