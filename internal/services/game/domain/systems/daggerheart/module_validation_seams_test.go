package daggerheart

import (
	"encoding/json"
	"testing"

	daggerheartvalidator "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/validator"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestValidationCommonHelpers(t *testing.T) {
	if err := daggerheartvalidator.RequireTrimmedValue(" ", "character_id"); err == nil {
		t.Fatal("expected required string validation error")
	}
	if err := daggerheartvalidator.RequireTrimmedValue("char-1", "character_id"); err != nil {
		t.Fatalf("daggerheartvalidator.RequireTrimmedValue(valid): %v", err)
	}
	if err := daggerheartvalidator.RequirePositive(0, "max"); err == nil {
		t.Fatal("expected positive validation error")
	}
	if err := daggerheartvalidator.RequirePositive(2, "max"); err != nil {
		t.Fatalf("daggerheartvalidator.RequirePositive(valid): %v", err)
	}
	if err := daggerheartvalidator.RequireRange(5, 0, 4, "marks"); err == nil {
		t.Fatal("expected range validation error")
	}
	if err := daggerheartvalidator.RequireRange(3, 0, 4, "marks"); err != nil {
		t.Fatalf("daggerheartvalidator.RequireRange(valid): %v", err)
	}
}

func TestValidateGMFearSetPayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateGMFearSetPayload(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected missing after error")
	}
	if err := daggerheartvalidator.ValidateGMFearSetPayload(json.RawMessage(`{"after":100}`)); err == nil {
		t.Fatal("expected out-of-range after error")
	}
	if err := daggerheartvalidator.ValidateGMFearSetPayload(json.RawMessage(`{"after":3}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateLoadoutSwapPayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateLoadoutSwapPayload(json.RawMessage(`{"character_id":" ","card_id":"c1"}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := daggerheartvalidator.ValidateLoadoutSwapPayload(json.RawMessage(`{"character_id":"char-1","card_id":" "}`)); err == nil {
		t.Fatal("expected missing card_id error")
	}
	if err := daggerheartvalidator.ValidateLoadoutSwapPayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","from":"active","to":"vault"}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateRestTakePayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateRestTakePayload(json.RawMessage(`{"rest_type":" "}`)); err == nil {
		t.Fatal("expected missing rest_type error")
	}
	if err := daggerheartvalidator.ValidateRestTakePayload(json.RawMessage(`{"rest_type":"short","participants":["char-1"],"campaign_countdown_advances":[{"countdown_id":"","before_remaining":1,"after_remaining":2,"advanced_by":1}]}`)); err == nil {
		t.Fatal("expected invalid campaign_countdown_advances error")
	}
	if err := daggerheartvalidator.ValidateRestTakePayload(json.RawMessage(`{"rest_type":"short","gm_fear_before":1,"gm_fear_after":1,"short_rests_before":1,"short_rests_after":1}`)); err == nil {
		t.Fatal("expected participants required error")
	}
	if err := daggerheartvalidator.ValidateRestTakePayload(json.RawMessage(`{"rest_type":"short","participants":["char-1"],"gm_fear_before":1,"gm_fear_after":2}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateCharacterTemporaryArmorApplyPayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":" ","source":"ritual","duration":"short_rest","amount":1}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := daggerheartvalidator.ValidateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":"char-1","source":" ","duration":"short_rest","amount":1}`)); err == nil {
		t.Fatal("expected missing source error")
	}
	if err := daggerheartvalidator.ValidateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":"char-1","source":"ritual","duration":"minute","amount":1}`)); err == nil {
		t.Fatal("expected invalid duration error")
	}
	if err := daggerheartvalidator.ValidateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":0}`)); err == nil {
		t.Fatal("expected amount-positive error")
	}
}

func TestValidateDowntimeMoveAppliedPayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":" ","move":"prepare","hope_after":2}`)); err == nil {
		t.Fatal("expected missing actor_character_id error")
	}
	if err := daggerheartvalidator.ValidateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","move":" "}`)); err == nil {
		t.Fatal("expected missing move error")
	}
	if err := daggerheartvalidator.ValidateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","move":"prepare"}`)); err == nil {
		t.Fatal("expected target-or-countdown error")
	}
	if err := daggerheartvalidator.ValidateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","target_character_id":"char-1","move":"prepare"}`)); err == nil {
		t.Fatal("expected missing state change error")
	}
	if err := daggerheartvalidator.ValidateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","target_character_id":"char-1","move":"prepare","hope_after":3}`)); err != nil {
		t.Fatalf("expected valid target state change payload, got: %v", err)
	}
	if err := daggerheartvalidator.ValidateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","move":"work_on_project","campaign_countdown_id":"cd-1"}`)); err != nil {
		t.Fatalf("expected valid countdown-only payload, got: %v", err)
	}
}

func TestValidateGMMoveTarget_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateGMMoveTarget(daggerheartpayload.GMMoveTarget{Type: "mystery"}); err == nil {
		t.Fatal("expected unsupported target type error")
	}
	if err := daggerheartvalidator.ValidateGMMoveTarget(daggerheartpayload.GMMoveTarget{Type: rules.GMMoveTargetTypeDirectMove, Kind: "interrupt_and_move", Shape: "custom"}); err == nil {
		t.Fatal("expected custom move description error")
	}
	if err := daggerheartvalidator.ValidateGMMoveTarget(daggerheartpayload.GMMoveTarget{Type: rules.GMMoveTargetTypeAdversaryFeature, FeatureID: "fear-feature"}); err == nil {
		t.Fatal("expected missing adversary_id error")
	}
	if err := daggerheartvalidator.ValidateGMMoveTarget(daggerheartpayload.GMMoveTarget{Type: rules.GMMoveTargetTypeEnvironmentFeature, EnvironmentID: "env-1"}); err == nil {
		t.Fatal("expected missing environment feature_id error")
	}
	if err := daggerheartvalidator.ValidateGMMoveTarget(daggerheartpayload.GMMoveTarget{Type: rules.GMMoveTargetTypeAdversaryExperience, AdversaryID: "adv-1"}); err == nil {
		t.Fatal("expected missing experience_name error")
	}
	if err := daggerheartvalidator.ValidateGMMoveTarget(daggerheartpayload.GMMoveTarget{
		Type:        rules.GMMoveTargetTypeDirectMove,
		Kind:        rules.GMMoveKindInterruptAndMove,
		Shape:       rules.GMMoveShapeRevealDanger,
		Description: "The bridge starts collapsing.",
	}); err != nil {
		t.Fatalf("expected valid direct move target, got: %v", err)
	}
}

func TestValidateDomainCardAcquirePayload_CardLevel(t *testing.T) {
	if err := daggerheartvalidator.ValidateDomainCardAcquirePayload(json.RawMessage(`{"character_id":" ","card_id":"card-1","card_level":1,"destination":"vault"}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := daggerheartvalidator.ValidateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":" ","card_level":1,"destination":"vault"}`)); err == nil {
		t.Fatal("expected missing card_id error")
	}
	if err := daggerheartvalidator.ValidateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":0,"destination":"vault"}`)); err == nil {
		t.Fatal("expected card_level < 1 error")
	}
	if err := daggerheartvalidator.ValidateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":-1,"destination":"vault"}`)); err == nil {
		t.Fatal("expected card_level negative error")
	}
	if err := daggerheartvalidator.ValidateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":1,"destination":"stash"}`)); err == nil {
		t.Fatal("expected invalid destination error")
	}
	if err := daggerheartvalidator.ValidateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":1,"destination":"vault"}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
	if err := daggerheartvalidator.ValidateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":3,"destination":"loadout"}`)); err != nil {
		t.Fatalf("expected valid payload with higher level, got: %v", err)
	}
}

func TestValidateSceneCountdownCreatePayload_SRDValidation(t *testing.T) {
	if err := daggerheartvalidator.ValidateSceneCountdownCreatePayload(json.RawMessage(`{"countdown_id":"cd-1","name":"Doom","tone":"progress","advancement_policy":"action_dynamic","starting_value":4,"remaining_value":4,"loop_behavior":"none","status":"active"}`)); err != nil {
		t.Fatalf("expected valid srd countdown payload, got: %v", err)
	}
	if err := daggerheartvalidator.ValidateSceneCountdownCreatePayload(json.RawMessage(`{"countdown_id":"cd-1","name":"Doom","tone":"chaos","advancement_policy":"action_dynamic","starting_value":4,"remaining_value":4,"loop_behavior":"none","status":"active"}`)); err == nil {
		t.Fatal("expected invalid tone rejection")
	}
	if err := daggerheartvalidator.ValidateSceneCountdownCreatePayload(json.RawMessage(`{"countdown_id":"cd-1","name":"Doom","tone":"progress","advancement_policy":"action_dynamic","starting_value":4,"remaining_value":4,"loop_behavior":"spiral","status":"active"}`)); err == nil {
		t.Fatal("expected invalid loop_behavior rejection")
	}
	if err := daggerheartvalidator.ValidateSceneCountdownCreatePayload(json.RawMessage(`{"countdown_id":"cd-1","name":"Doom","tone":"progress","advancement_policy":"action_dynamic","starting_value":4,"remaining_value":4,"loop_behavior":"none","status":"active","starting_roll":{"min":6,"max":4,"value":5}}`)); err == nil {
		t.Fatal("expected invalid starting_roll range rejection")
	}
	if err := daggerheartvalidator.ValidateSceneCountdownCreatePayload(json.RawMessage(`{"countdown_id":"cd-1","name":"Doom","tone":"progress","advancement_policy":"action_dynamic","starting_value":4,"remaining_value":4,"loop_behavior":"none","status":"trigger_pending","starting_roll":{"min":1,"max":6,"value":7}}`)); err == nil {
		t.Fatal("expected invalid starting_roll value rejection")
	}
}

func TestValidateGoldUpdatePayload_Branches(t *testing.T) {
	// No-mutation rejected.
	if err := daggerheartvalidator.ValidateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":3,"handfuls_after":3,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":0}`)); err == nil {
		t.Fatal("expected no-mutation rejection")
	}
	// Handfuls out of range.
	if err := daggerheartvalidator.ValidateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":0,"handfuls_after":10,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":0}`)); err == nil {
		t.Fatal("expected handfuls range error")
	}
	// Bags out of range.
	if err := daggerheartvalidator.ValidateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":0,"handfuls_after":0,"bags_before":0,"bags_after":10,"chests_before":0,"chests_after":0}`)); err == nil {
		t.Fatal("expected bags range error")
	}
	// Chests out of range.
	if err := daggerheartvalidator.ValidateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":0,"handfuls_after":0,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":2}`)); err == nil {
		t.Fatal("expected chests range error")
	}
	// Valid.
	if err := daggerheartvalidator.ValidateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":0,"handfuls_after":3,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":0}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateEquipmentSwapPayload_Branches(t *testing.T) {
	// Invalid item_type.
	if err := daggerheartvalidator.ValidateEquipmentSwapPayload(json.RawMessage(`{"character_id":"char-1","item_id":"s1","item_type":"shield","from":"inventory","to":"active"}`)); err == nil {
		t.Fatal("expected invalid item_type error")
	}
	// Invalid slot.
	if err := daggerheartvalidator.ValidateEquipmentSwapPayload(json.RawMessage(`{"character_id":"char-1","item_id":"s1","item_type":"weapon","from":"backpack","to":"active"}`)); err == nil {
		t.Fatal("expected invalid slot error")
	}
	// From == to.
	if err := daggerheartvalidator.ValidateEquipmentSwapPayload(json.RawMessage(`{"character_id":"char-1","item_id":"s1","item_type":"weapon","from":"active","to":"active"}`)); err == nil {
		t.Fatal("expected from==to error")
	}
	// Valid.
	if err := daggerheartvalidator.ValidateEquipmentSwapPayload(json.RawMessage(`{"character_id":"char-1","item_id":"s1","item_type":"armor","from":"inventory","to":"active"}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateConsumablePayloads_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateConsumableUsePayload(json.RawMessage(`{"character_id":" ","consumable_id":"p1","quantity_before":2,"quantity_after":1}`)); err == nil {
		t.Fatal("expected use missing character_id error")
	}
	if err := daggerheartvalidator.ValidateConsumableUsePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":" ","quantity_before":2,"quantity_after":1}`)); err == nil {
		t.Fatal("expected use missing consumable_id error")
	}
	// Use: quantity_before must be positive.
	if err := daggerheartvalidator.ValidateConsumableUsePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":0,"quantity_after":-1}`)); err == nil {
		t.Fatal("expected use quantity_before positive error")
	}
	// Use: quantity_after must be quantity_before - 1.
	if err := daggerheartvalidator.ValidateConsumableUsePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":2,"quantity_after":0}`)); err == nil {
		t.Fatal("expected use quantity_after mismatch error")
	}
	// Use: valid.
	if err := daggerheartvalidator.ValidateConsumableUsePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":2,"quantity_after":1}`)); err != nil {
		t.Fatalf("expected valid use payload, got: %v", err)
	}
	// Acquire: missing character_id / consumable_id.
	if err := daggerheartvalidator.ValidateConsumableAcquirePayload(json.RawMessage(`{"character_id":" ","consumable_id":"p1","quantity_before":1,"quantity_after":2}`)); err == nil {
		t.Fatal("expected acquire missing character_id error")
	}
	if err := daggerheartvalidator.ValidateConsumableAcquirePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":" ","quantity_before":1,"quantity_after":2}`)); err == nil {
		t.Fatal("expected acquire missing consumable_id error")
	}
	// Acquire: quantity_after out of range.
	if err := daggerheartvalidator.ValidateConsumableAcquirePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":5,"quantity_after":6}`)); err == nil {
		t.Fatal("expected acquire quantity_after out of range error")
	}
	// Acquire: quantity_after != quantity_before + 1.
	if err := daggerheartvalidator.ValidateConsumableAcquirePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":1,"quantity_after":3}`)); err == nil {
		t.Fatal("expected acquire quantity_after mismatch error")
	}
	// Acquire: valid.
	if err := daggerheartvalidator.ValidateConsumableAcquirePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":1,"quantity_after":2}`)); err != nil {
		t.Fatalf("expected valid acquire payload, got: %v", err)
	}
}

func TestValidateLevelUpApplyPayload_Branches(t *testing.T) {
	// Missing advancements.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[]}`)); err == nil {
		t.Fatal("expected empty advancements error")
	}
	// level_after != level_before + 1.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":3,"advancements":[{"type":"add_hp_slots"}]}`)); err == nil {
		t.Fatal("expected level_after != level_before+1 error")
	}
	// Valid.
	if err := daggerheartvalidator.ValidateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}]}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateAdversaryCreateUpdatePayload_Branches(t *testing.T) {
	if err := daggerheartvalidator.ValidateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":" ","name":"Goblin"}`)); err == nil {
		t.Fatal("expected missing adversary_id error")
	}
	if err := daggerheartvalidator.ValidateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":" ","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`)); err == nil {
		t.Fatal("expected missing adversary_entry_id error")
	}
	if err := daggerheartvalidator.ValidateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","name":" "}`)); err == nil {
		t.Fatal("expected missing name error")
	}
	if err := daggerheartvalidator.ValidateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":"entry-1","name":"Goblin","session_id":" ","scene_id":"scene-1"}`)); err == nil {
		t.Fatal("expected missing session_id error")
	}
	if err := daggerheartvalidator.ValidateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":"entry-1","name":"Goblin","session_id":"sess-1","scene_id":" "}`)); err == nil {
		t.Fatal("expected missing scene_id error")
	}
	if err := daggerheartvalidator.ValidateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":"entry-1","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`)); err != nil {
		t.Fatalf("expected valid create payload, got: %v", err)
	}
	if err := daggerheartvalidator.ValidateAdversaryUpdatePayload(json.RawMessage(`{"adversary_id":"adv-1","name":" "}`)); err == nil {
		t.Fatal("expected update missing name error")
	}
	if err := daggerheartvalidator.ValidateAdversaryUpdatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":"entry-1","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`)); err != nil {
		t.Fatalf("expected valid update payload, got: %v", err)
	}
	if err := daggerheartvalidator.ValidateAdversaryDeletePayload(json.RawMessage(`{"adversary_id":" "}`)); err == nil {
		t.Fatal("expected delete missing adversary_id error")
	}
	if err := daggerheartvalidator.ValidateAdversaryDeletePayload(json.RawMessage(`{"adversary_id":"adv-1"}`)); err != nil {
		t.Fatalf("expected valid delete payload, got: %v", err)
	}
}
