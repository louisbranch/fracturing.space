package daggerheart

import (
	"encoding/json"
	"testing"

	daggerheartvalidator "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/validator"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestHasSubclassStateFieldChange(t *testing.T) {
	// Both nil → no change.
	if daggerheartvalidator.HasSubclassStateFieldChange(nil, nil) {
		t.Fatal("nil/nil should be no change")
	}
	// One nil → change.
	state := &daggerheartstate.CharacterSubclassState{}
	if !daggerheartvalidator.HasSubclassStateFieldChange(nil, state) {
		t.Fatal("nil/non-nil should be change")
	}
	if !daggerheartvalidator.HasSubclassStateFieldChange(state, nil) {
		t.Fatal("non-nil/nil should be change")
	}
	// Same values → no change.
	a := &daggerheartstate.CharacterSubclassState{ElementalChannel: "fire"}
	b := &daggerheartstate.CharacterSubclassState{ElementalChannel: "fire"}
	if daggerheartvalidator.HasSubclassStateFieldChange(a, b) {
		t.Fatal("equal states should be no change")
	}
	// Different values → change.
	c := &daggerheartstate.CharacterSubclassState{ElementalChannel: "water"}
	if !daggerheartvalidator.HasSubclassStateFieldChange(a, c) {
		t.Fatal("different states should be change")
	}
}

func TestHasCompanionStateFieldChange(t *testing.T) {
	// Both nil → no change.
	if daggerheartvalidator.HasCompanionStateFieldChange(nil, nil) {
		t.Fatal("nil/nil should be no change")
	}
	// One nil → change.
	state := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent}
	if !daggerheartvalidator.HasCompanionStateFieldChange(nil, state) {
		t.Fatal("nil/non-nil should be change")
	}
	if !daggerheartvalidator.HasCompanionStateFieldChange(state, nil) {
		t.Fatal("non-nil/nil should be change")
	}
	// Same values → no change.
	a := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusAway, ActiveExperienceID: "exp-1"}
	b := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusAway, ActiveExperienceID: "exp-1"}
	if daggerheartvalidator.HasCompanionStateFieldChange(a, b) {
		t.Fatal("equal states should be no change")
	}
	// Different values → change.
	c := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent}
	if !daggerheartvalidator.HasCompanionStateFieldChange(a, c) {
		t.Fatal("different states should be change")
	}
}

func TestHasBoolFieldChange(t *testing.T) {
	// nil after → no change.
	if daggerheartvalidator.HasBoolFieldChange(nil, nil) {
		t.Fatal("nil/nil should be no change")
	}
	trueVal := true
	falseVal := false
	// nil before, non-nil after → change.
	if !daggerheartvalidator.HasBoolFieldChange(nil, &trueVal) {
		t.Fatal("nil/true should be change")
	}
	// Same values → no change.
	if daggerheartvalidator.HasBoolFieldChange(&trueVal, &trueVal) {
		t.Fatal("true/true should be no change")
	}
	// Different values → change.
	if !daggerheartvalidator.HasBoolFieldChange(&trueVal, &falseVal) {
		t.Fatal("true/false should be change")
	}
}

func TestHasConditionListMutation_ErrorPaths(t *testing.T) {
	// Same lists → no mutation.
	if daggerheartvalidator.HasConditionListMutation([]string{"hidden"}, []string{"hidden"}) {
		t.Fatal("same lists should be no mutation")
	}
	// Different lists → mutation.
	if !daggerheartvalidator.HasConditionListMutation([]string{"hidden"}, []string{"hidden", "restrained"}) {
		t.Fatal("different lists should be mutation")
	}
	// Invalid condition triggers mutation (error path returns true).
	if !daggerheartvalidator.HasConditionListMutation([]string{"invalid"}, []string{"hidden"}) {
		t.Fatal("invalid before list should trigger mutation")
	}
	if !daggerheartvalidator.HasConditionListMutation([]string{"hidden"}, []string{"invalid"}) {
		t.Fatal("invalid after list should trigger mutation")
	}
}

func TestValidateCharacterStatePatchPayload_Branches(t *testing.T) {
	// Missing character_id.
	if err := daggerheartvalidator.ValidateCharacterStatePatchPayload(json.RawMessage(`{"character_id":" "}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	// No change field.
	if err := daggerheartvalidator.ValidateCharacterStatePatchPayload(json.RawMessage(`{"character_id":"char-1"}`)); err == nil {
		t.Fatal("expected no-change error")
	}
	// Valid with HP change.
	if err := daggerheartvalidator.ValidateCharacterStatePatchPayload(json.RawMessage(`{"character_id":"char-1","hp_before":5,"hp_after":3}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
	// Valid with life state change.
	if err := daggerheartvalidator.ValidateCharacterStatePatchPayload(json.RawMessage(`{"character_id":"char-1","life_state_before":"alive","life_state_after":"dead"}`)); err != nil {
		t.Fatalf("expected valid life state payload, got: %v", err)
	}
	// Valid with impenetrable change.
	if err := daggerheartvalidator.ValidateCharacterStatePatchPayload(json.RawMessage(`{"character_id":"char-1","impenetrable_used_this_short_rest_before":false,"impenetrable_used_this_short_rest_after":true}`)); err != nil {
		t.Fatalf("expected valid impenetrable payload, got: %v", err)
	}
}

func TestValidateCharacterStatePatchedPayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateCharacterStatePatchedPayload(json.RawMessage(`{"character_id":" "}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := daggerheartvalidator.ValidateCharacterStatePatchedPayload(json.RawMessage(`{"character_id":"char-1"}`)); err == nil {
		t.Fatal("expected no-field error")
	}
	if err := daggerheartvalidator.ValidateCharacterStatePatchedPayload(json.RawMessage(`{"character_id":"char-1","hp_after":5}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateSubclassFeatureApplyPayload_Branches(t *testing.T) {
	// Missing actor.
	if err := daggerheartvalidator.ValidateSubclassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":" ","feature":"test"}`)); err == nil {
		t.Fatal("expected missing actor error")
	}
	// Missing feature.
	if err := daggerheartvalidator.ValidateSubclassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":" "}`)); err == nil {
		t.Fatal("expected missing feature error")
	}
	// No targets at all.
	if err := daggerheartvalidator.ValidateSubclassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":"test"}`)); err == nil {
		t.Fatal("expected no-target error")
	}
	// Targets with no mutation.
	if err := daggerheartvalidator.ValidateSubclassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":"test","targets":[{"character_id":"char-2"}]}`)); err == nil {
		t.Fatal("expected no-mutation error")
	}
	// Valid with character condition target.
	if err := daggerheartvalidator.ValidateSubclassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":"test","character_condition_targets":[{"character_id":"char-2","conditions_after":[{"id":"hidden","class":"standard","standard":"hidden","code":"hidden","label":"Hidden"}]}]}`)); err != nil {
		t.Fatalf("expected valid condition target payload, got: %v", err)
	}
	// Valid with subclass state change.
	if err := daggerheartvalidator.ValidateSubclassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":"test","targets":[{"character_id":"char-2","subclass_state_after":{"battle_ritual_used_this_long_rest":true}}]}`)); err != nil {
		t.Fatalf("expected valid subclass state target payload, got: %v", err)
	}
}

func TestValidateAdversaryFeatureApplyPayload_Branches(t *testing.T) {
	// No mutation.
	if err := daggerheartvalidator.ValidateAdversaryFeatureApplyPayload(json.RawMessage(`{"actor_adversary_id":"adv-1","adversary_id":"adv-1","feature_id":"f1"}`)); err == nil {
		t.Fatal("expected no-mutation error")
	}
	// Valid with stress change.
	if err := daggerheartvalidator.ValidateAdversaryFeatureApplyPayload(json.RawMessage(`{"actor_adversary_id":"adv-1","adversary_id":"adv-1","feature_id":"f1","stress_before":0,"stress_after":2}`)); err != nil {
		t.Fatalf("expected valid stress payload, got: %v", err)
	}
	// Valid with feature state change.
	if err := daggerheartvalidator.ValidateAdversaryFeatureApplyPayload(json.RawMessage(`{"actor_adversary_id":"adv-1","adversary_id":"adv-1","feature_id":"f1","feature_states_before":[],"feature_states_after":[{"feature_id":"f1","status":"used"}]}`)); err != nil {
		t.Fatalf("expected valid feature state payload, got: %v", err)
	}
	// Valid with pending experience change.
	if err := daggerheartvalidator.ValidateAdversaryFeatureApplyPayload(json.RawMessage(`{"actor_adversary_id":"adv-1","adversary_id":"adv-1","feature_id":"f1","pending_experience_after":{"name":"xp","modifier":10}}`)); err != nil {
		t.Fatalf("expected valid pending xp payload, got: %v", err)
	}
}

func TestValidateHopeSpendPayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateHopeSpendPayload(json.RawMessage(`{"character_id":" ","amount":1,"before":3,"after":2}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := daggerheartvalidator.ValidateHopeSpendPayload(json.RawMessage(`{"character_id":"char-1","amount":0,"before":3,"after":3}`)); err == nil {
		t.Fatal("expected non-positive amount error")
	}
	if err := daggerheartvalidator.ValidateHopeSpendPayload(json.RawMessage(`{"character_id":"char-1","amount":1,"before":3,"after":3}`)); err == nil {
		t.Fatal("expected before==after error")
	}
	if err := daggerheartvalidator.ValidateHopeSpendPayload(json.RawMessage(`{"character_id":"char-1","amount":2,"before":3,"after":2}`)); err == nil {
		t.Fatal("expected delta mismatch error")
	}
	if err := daggerheartvalidator.ValidateHopeSpendPayload(json.RawMessage(`{"character_id":"char-1","amount":1,"before":3,"after":2}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateStressSpendPayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateStressSpendPayload(json.RawMessage(`{"character_id":" ","amount":1,"before":3,"after":2}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := daggerheartvalidator.ValidateStressSpendPayload(json.RawMessage(`{"character_id":"char-1","amount":0,"before":3,"after":3}`)); err == nil {
		t.Fatal("expected non-positive amount error")
	}
	if err := daggerheartvalidator.ValidateStressSpendPayload(json.RawMessage(`{"character_id":"char-1","amount":1,"before":3,"after":2}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateBeastformPayloads_Branches(t *testing.T) {
	// Transform: missing fields.
	if err := daggerheartvalidator.ValidateBeastformTransformPayload(json.RawMessage(`{"actor_character_id":" ","character_id":"char-1","beastform_id":"bf-1","class_state_after":{}}`)); err == nil {
		t.Fatal("expected transform missing actor error")
	}
	if err := daggerheartvalidator.ValidateBeastformTransformPayload(json.RawMessage(`{"actor_character_id":"char-1","character_id":" ","beastform_id":"bf-1","class_state_after":{}}`)); err == nil {
		t.Fatal("expected transform missing character error")
	}
	if err := daggerheartvalidator.ValidateBeastformTransformPayload(json.RawMessage(`{"actor_character_id":"char-1","character_id":"char-1","beastform_id":" ","class_state_after":{}}`)); err == nil {
		t.Fatal("expected transform missing beastform error")
	}
	if err := daggerheartvalidator.ValidateBeastformTransformPayload(json.RawMessage(`{"actor_character_id":"char-1","character_id":"char-1","beastform_id":"bf-1"}`)); err == nil {
		t.Fatal("expected transform no-change error")
	}

	// Drop: missing fields.
	if err := daggerheartvalidator.ValidateBeastformDropPayload(json.RawMessage(`{"actor_character_id":" ","character_id":"char-1","beastform_id":"bf-1","class_state_after":{}}`)); err == nil {
		t.Fatal("expected drop missing actor error")
	}
	if err := daggerheartvalidator.ValidateBeastformDropPayload(json.RawMessage(`{"actor_character_id":"char-1","character_id":"char-1","beastform_id":"bf-1"}`)); err == nil {
		t.Fatal("expected drop no-class-state error")
	}
}

func TestValidateCompanionPayloads_Branches(t *testing.T) {
	// Begin: missing fields.
	if err := daggerheartvalidator.ValidateCompanionExperienceBeginPayload(json.RawMessage(`{"actor_character_id":" ","character_id":"char-1","experience_id":"exp-1","companion_state_after":{"status":"away","active_experience_id":"exp-1"}}`)); err == nil {
		t.Fatal("expected begin missing actor error")
	}
	if err := daggerheartvalidator.ValidateCompanionExperienceBeginPayload(json.RawMessage(`{"actor_character_id":"char-1","character_id":" ","experience_id":"exp-1","companion_state_after":{"status":"away","active_experience_id":"exp-1"}}`)); err == nil {
		t.Fatal("expected begin missing character error")
	}
	if err := daggerheartvalidator.ValidateCompanionExperienceBeginPayload(json.RawMessage(`{"actor_character_id":"char-1","character_id":"char-1","experience_id":" ","companion_state_after":{"status":"away","active_experience_id":"exp-1"}}`)); err == nil {
		t.Fatal("expected begin missing experience error")
	}
	if err := daggerheartvalidator.ValidateCompanionExperienceBeginPayload(json.RawMessage(`{"actor_character_id":"char-1","character_id":"char-1","experience_id":"exp-1"}`)); err == nil {
		t.Fatal("expected begin no-state-change error")
	}

	// Return: missing fields.
	if err := daggerheartvalidator.ValidateCompanionReturnPayload(json.RawMessage(`{"actor_character_id":" ","character_id":"char-1","resolution":"success","companion_state_after":{"status":"present"}}`)); err == nil {
		t.Fatal("expected return missing actor error")
	}
	if err := daggerheartvalidator.ValidateCompanionReturnPayload(json.RawMessage(`{"actor_character_id":"char-1","character_id":"char-1","resolution":" ","companion_state_after":{"status":"present"}}`)); err == nil {
		t.Fatal("expected return missing resolution error")
	}
	if err := daggerheartvalidator.ValidateCompanionReturnPayload(json.RawMessage(`{"actor_character_id":"char-1","character_id":"char-1","resolution":"success"}`)); err == nil {
		t.Fatal("expected return no-change error")
	}
}

func TestValidateEnvironmentEntityPayloads_Branches(t *testing.T) {
	// Create: tier and difficulty validation.
	if err := daggerheartvalidator.ValidateEnvironmentEntityCreatePayload(json.RawMessage(`{"environment_entity_id":"ee-1","environment_id":"env-1","name":"Trap","type":"hazard","session_id":"sess-1","scene_id":"scene-1","tier":-1,"difficulty":5}`)); err == nil {
		t.Fatal("expected negative tier error")
	}
	if err := daggerheartvalidator.ValidateEnvironmentEntityCreatePayload(json.RawMessage(`{"environment_entity_id":"ee-1","environment_id":"env-1","name":"Trap","type":"hazard","session_id":"sess-1","scene_id":"scene-1","tier":1,"difficulty":0}`)); err == nil {
		t.Fatal("expected non-positive difficulty error")
	}
	// Valid.
	if err := daggerheartvalidator.ValidateEnvironmentEntityCreatePayload(json.RawMessage(`{"environment_entity_id":"ee-1","environment_id":"env-1","name":"Trap","type":"hazard","session_id":"sess-1","scene_id":"scene-1","tier":1,"difficulty":5}`)); err != nil {
		t.Fatalf("expected valid create payload, got: %v", err)
	}

	// Delete: missing environment_entity_id.
	if err := daggerheartvalidator.ValidateEnvironmentEntityDeletePayload(json.RawMessage(`{"environment_entity_id":" "}`)); err == nil {
		t.Fatal("expected missing environment_entity_id error")
	}
}

func TestEqualAdversaryFeatureStates(t *testing.T) {
	if !daggerheartvalidator.EqualAdversaryFeatureStates(nil, nil) {
		t.Fatal("nil/nil should be equal")
	}
	a := []rules.AdversaryFeatureState{{FeatureID: "f1", Status: "active"}}
	if daggerheartvalidator.EqualAdversaryFeatureStates(a, nil) {
		t.Fatal("non-nil/nil should not be equal")
	}
	b := []rules.AdversaryFeatureState{{FeatureID: "f1", Status: "active"}}
	if !daggerheartvalidator.EqualAdversaryFeatureStates(a, b) {
		t.Fatal("same values should be equal")
	}
	c := []rules.AdversaryFeatureState{{FeatureID: "f1", Status: "used"}}
	if daggerheartvalidator.EqualAdversaryFeatureStates(a, c) {
		t.Fatal("different status should not be equal")
	}
}

func TestEqualAdversaryPendingExperience(t *testing.T) {
	if !daggerheartvalidator.EqualAdversaryPendingExperience(nil, nil) {
		t.Fatal("nil/nil should be equal")
	}
	a := &rules.AdversaryPendingExperience{Name: "xp", Modifier: 10}
	if daggerheartvalidator.EqualAdversaryPendingExperience(a, nil) {
		t.Fatal("non-nil/nil should not be equal")
	}
	if daggerheartvalidator.EqualAdversaryPendingExperience(nil, a) {
		t.Fatal("nil/non-nil should not be equal")
	}
	b := &rules.AdversaryPendingExperience{Name: "xp", Modifier: 10}
	if !daggerheartvalidator.EqualAdversaryPendingExperience(a, b) {
		t.Fatal("same values should be equal")
	}
	c := &rules.AdversaryPendingExperience{Name: "xp", Modifier: 20}
	if daggerheartvalidator.EqualAdversaryPendingExperience(a, c) {
		t.Fatal("different modifier should not be equal")
	}
}

func TestValidateRestTakePayload_DowntimeMoveValidation(t *testing.T) {
	// Valid with downtime moves.
	payload := `{"rest_type":"short","participants":["char-1"],"gm_fear_before":1,"gm_fear_after":1,"downtime_moves":[{"actor_character_id":"char-1","target_character_id":"char-2","move":"prepare","hope_after":3}]}`
	if err := daggerheartvalidator.ValidateRestTakePayload(json.RawMessage(payload)); err != nil {
		t.Fatalf("expected valid payload with downtime moves, got: %v", err)
	}
}

func TestValidateLevelUpApplyPayload_ExtraBranches(t *testing.T) {
	// level_before out of range.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":0,"level_after":1,"advancements":[{"type":"add_hp_slots"}]}`)); err == nil {
		t.Fatal("expected level_before out of range error")
	}
	// level_after out of range.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":10,"level_after":11,"advancements":[{"type":"add_hp_slots"}]}`)); err == nil {
		t.Fatal("expected level_after out of range error")
	}
	// missing character_id.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":" ","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}]}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	// reward domain_card missing card_id.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}],"rewards":[{"type":"domain_card","domain_card_id":"","domain_card_level":1}]}`)); err == nil {
		t.Fatal("expected reward missing card_id error")
	}
	// reward domain_card invalid card_level.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}],"rewards":[{"type":"domain_card","domain_card_id":"card-1","domain_card_level":0}]}`)); err == nil {
		t.Fatal("expected reward invalid card_level error")
	}
	// reward companion_bonus_choices invalid.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}],"rewards":[{"type":"companion_bonus_choices","companion_bonus_choices":0}]}`)); err == nil {
		t.Fatal("expected reward invalid companion_bonus_choices error")
	}
	// reward missing type.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}],"rewards":[{"type":""}]}`)); err == nil {
		t.Fatal("expected reward missing type error")
	}
	// reward unsupported type.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}],"rewards":[{"type":"unknown"}]}`)); err == nil {
		t.Fatal("expected reward unsupported type error")
	}
	// valid with domain_card reward.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}],"rewards":[{"type":"domain_card","domain_card_id":"card-1","domain_card_level":2}]}`)); err != nil {
		t.Fatalf("expected valid level up with reward, got: %v", err)
	}
}

func TestValidateEnvironmentEntityUpdatePayload_Branches(t *testing.T) {
	// Missing required fields.
	if err := daggerheartvalidator.ValidateEnvironmentEntityUpdatePayload(json.RawMessage(`{"environment_entity_id":" "}`)); err == nil {
		t.Fatal("expected missing entity_id error")
	}
	if err := daggerheartvalidator.ValidateEnvironmentEntityUpdatePayload(json.RawMessage(`{"environment_entity_id":"ee-1","environment_id":" "}`)); err == nil {
		t.Fatal("expected missing environment_id error")
	}
	if err := daggerheartvalidator.ValidateEnvironmentEntityUpdatePayload(json.RawMessage(`{"environment_entity_id":"ee-1","environment_id":"env-1","name":" "}`)); err == nil {
		t.Fatal("expected missing name error")
	}
	if err := daggerheartvalidator.ValidateEnvironmentEntityUpdatePayload(json.RawMessage(`{"environment_entity_id":"ee-1","environment_id":"env-1","name":"Trap","type":" "}`)); err == nil {
		t.Fatal("expected missing type error")
	}
	if err := daggerheartvalidator.ValidateEnvironmentEntityUpdatePayload(json.RawMessage(`{"environment_entity_id":"ee-1","environment_id":"env-1","name":"Trap","type":"hazard","session_id":" ","scene_id":"scene-1","tier":1,"difficulty":5}`)); err == nil {
		t.Fatal("expected missing session_id error")
	}
	if err := daggerheartvalidator.ValidateEnvironmentEntityUpdatePayload(json.RawMessage(`{"environment_entity_id":"ee-1","environment_id":"env-1","name":"Trap","type":"hazard","session_id":"sess-1","scene_id":" ","tier":1,"difficulty":5}`)); err == nil {
		t.Fatal("expected missing scene_id error")
	}
	// Valid.
	if err := daggerheartvalidator.ValidateEnvironmentEntityUpdatePayload(json.RawMessage(`{"environment_entity_id":"ee-1","environment_id":"env-1","name":"Trap","type":"hazard","session_id":"sess-1","scene_id":"scene-1","tier":1,"difficulty":5}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateClassFeatureApplyPayload_Branches(t *testing.T) {
	// Missing actor.
	if err := daggerheartvalidator.ValidateClassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":" ","feature":"shield_wall","targets":[{"character_id":"char-1","hp_before":5,"hp_after":3}]}`)); err == nil {
		t.Fatal("expected missing actor error")
	}
	// Missing feature.
	if err := daggerheartvalidator.ValidateClassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":" ","targets":[{"character_id":"char-1","hp_before":5,"hp_after":3}]}`)); err == nil {
		t.Fatal("expected missing feature error")
	}
	// No targets.
	if err := daggerheartvalidator.ValidateClassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":"shield_wall","targets":[]}`)); err == nil {
		t.Fatal("expected no-target error")
	}
	// Target without character_id.
	if err := daggerheartvalidator.ValidateClassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":"shield_wall","targets":[{"character_id":" ","hp_before":5,"hp_after":3}]}`)); err == nil {
		t.Fatal("expected target missing character_id error")
	}
	// Target without field changes.
	if err := daggerheartvalidator.ValidateClassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":"shield_wall","targets":[{"character_id":"char-2"}]}`)); err == nil {
		t.Fatal("expected target no-change error")
	}
	// Valid.
	if err := daggerheartvalidator.ValidateClassFeatureApplyPayload(json.RawMessage(`{"actor_character_id":"char-1","feature":"shield_wall","targets":[{"character_id":"char-2","hp_before":5,"hp_after":3}]}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestAbs(t *testing.T) {
	if daggerheartvalidator.Abs(-5) != 5 {
		t.Fatal("daggerheartvalidator.Abs(-5) should be 5")
	}
	if daggerheartvalidator.Abs(5) != 5 {
		t.Fatal("daggerheartvalidator.Abs(5) should be 5")
	}
	if daggerheartvalidator.Abs(0) != 0 {
		t.Fatal("daggerheartvalidator.Abs(0) should be 0")
	}
}
