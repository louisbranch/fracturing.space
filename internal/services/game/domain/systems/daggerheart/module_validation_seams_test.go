package daggerheart

import (
	"encoding/json"
	"testing"
)

func TestValidationCommonHelpers(t *testing.T) {
	if err := requireTrimmedValue(" ", "character_id"); err == nil {
		t.Fatal("expected required string validation error")
	}
	if err := requireTrimmedValue("char-1", "character_id"); err != nil {
		t.Fatalf("requireTrimmedValue(valid): %v", err)
	}
	if err := requirePositive(0, "max"); err == nil {
		t.Fatal("expected positive validation error")
	}
	if err := requirePositive(2, "max"); err != nil {
		t.Fatalf("requirePositive(valid): %v", err)
	}
	if err := requireRange(5, 0, 4, "marks"); err == nil {
		t.Fatal("expected range validation error")
	}
	if err := requireRange(3, 0, 4, "marks"); err != nil {
		t.Fatalf("requireRange(valid): %v", err)
	}
}

func TestValidateGMFearSetPayload_Branches(t *testing.T) {
	if err := validateGMFearSetPayload(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected missing after error")
	}
	if err := validateGMFearSetPayload(json.RawMessage(`{"after":100}`)); err == nil {
		t.Fatal("expected out-of-range after error")
	}
	if err := validateGMFearSetPayload(json.RawMessage(`{"after":3}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateLoadoutSwapPayload_Branches(t *testing.T) {
	if err := validateLoadoutSwapPayload(json.RawMessage(`{"character_id":" ","card_id":"c1"}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := validateLoadoutSwapPayload(json.RawMessage(`{"character_id":"char-1","card_id":" "}`)); err == nil {
		t.Fatal("expected missing card_id error")
	}
	if err := validateLoadoutSwapPayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","from":"active","to":"vault"}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateRestTakePayload_Branches(t *testing.T) {
	if err := validateRestTakePayload(json.RawMessage(`{"rest_type":" "}`)); err == nil {
		t.Fatal("expected missing rest_type error")
	}
	if err := validateRestTakePayload(json.RawMessage(`{"rest_type":"short","participants":["char-1"],"countdown_updates":[{"countdown_id":"","before":1,"after":2}]}`)); err == nil {
		t.Fatal("expected invalid countdown_updates error")
	}
	if err := validateRestTakePayload(json.RawMessage(`{"rest_type":"short","gm_fear_before":1,"gm_fear_after":1,"short_rests_before":1,"short_rests_after":1}`)); err == nil {
		t.Fatal("expected participants required error")
	}
	if err := validateRestTakePayload(json.RawMessage(`{"rest_type":"short","participants":["char-1"],"gm_fear_before":1,"gm_fear_after":2}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateCharacterTemporaryArmorApplyPayload_Branches(t *testing.T) {
	if err := validateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":" ","source":"ritual","duration":"short_rest","amount":1}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := validateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":"char-1","source":" ","duration":"short_rest","amount":1}`)); err == nil {
		t.Fatal("expected missing source error")
	}
	if err := validateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":"char-1","source":"ritual","duration":"minute","amount":1}`)); err == nil {
		t.Fatal("expected invalid duration error")
	}
	if err := validateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":0}`)); err == nil {
		t.Fatal("expected amount-positive error")
	}
}

func TestValidateDowntimeMoveAppliedPayload_Branches(t *testing.T) {
	if err := validateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":" ","move":"prepare","hope_after":2}`)); err == nil {
		t.Fatal("expected missing actor_character_id error")
	}
	if err := validateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","move":" "}`)); err == nil {
		t.Fatal("expected missing move error")
	}
	if err := validateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","move":"prepare"}`)); err == nil {
		t.Fatal("expected target-or-countdown error")
	}
	if err := validateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","target_character_id":"char-1","move":"prepare"}`)); err == nil {
		t.Fatal("expected missing state change error")
	}
	if err := validateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","target_character_id":"char-1","move":"prepare","hope_after":3}`)); err != nil {
		t.Fatalf("expected valid target state change payload, got: %v", err)
	}
	if err := validateDowntimeMoveAppliedPayload(json.RawMessage(`{"actor_character_id":"char-1","move":"work_on_project","countdown_id":"cd-1"}`)); err != nil {
		t.Fatalf("expected valid countdown-only payload, got: %v", err)
	}
}

func TestValidateGMMoveTarget_Branches(t *testing.T) {
	if err := validateGMMoveTarget(GMMoveTarget{Type: "mystery"}); err == nil {
		t.Fatal("expected unsupported target type error")
	}
	if err := validateGMMoveTarget(GMMoveTarget{Type: GMMoveTargetTypeDirectMove, Kind: "interrupt_and_move", Shape: "custom"}); err == nil {
		t.Fatal("expected custom move description error")
	}
	if err := validateGMMoveTarget(GMMoveTarget{Type: GMMoveTargetTypeAdversaryFeature, FeatureID: "fear-feature"}); err == nil {
		t.Fatal("expected missing adversary_id error")
	}
	if err := validateGMMoveTarget(GMMoveTarget{Type: GMMoveTargetTypeEnvironmentFeature, EnvironmentID: "env-1"}); err == nil {
		t.Fatal("expected missing environment feature_id error")
	}
	if err := validateGMMoveTarget(GMMoveTarget{Type: GMMoveTargetTypeAdversaryExperience, AdversaryID: "adv-1"}); err == nil {
		t.Fatal("expected missing experience_name error")
	}
	if err := validateGMMoveTarget(GMMoveTarget{
		Type:        GMMoveTargetTypeDirectMove,
		Kind:        GMMoveKindInterruptAndMove,
		Shape:       GMMoveShapeRevealDanger,
		Description: "The bridge starts collapsing.",
	}); err != nil {
		t.Fatalf("expected valid direct move target, got: %v", err)
	}
}

func TestValidateDomainCardAcquirePayload_CardLevel(t *testing.T) {
	if err := validateDomainCardAcquirePayload(json.RawMessage(`{"character_id":" ","card_id":"card-1","card_level":1,"destination":"vault"}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := validateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":" ","card_level":1,"destination":"vault"}`)); err == nil {
		t.Fatal("expected missing card_id error")
	}
	if err := validateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":0,"destination":"vault"}`)); err == nil {
		t.Fatal("expected card_level < 1 error")
	}
	if err := validateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":-1,"destination":"vault"}`)); err == nil {
		t.Fatal("expected card_level negative error")
	}
	if err := validateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":1,"destination":"stash"}`)); err == nil {
		t.Fatal("expected invalid destination error")
	}
	if err := validateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":1,"destination":"vault"}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
	if err := validateDomainCardAcquirePayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","card_level":3,"destination":"loadout"}`)); err != nil {
		t.Fatalf("expected valid payload with higher level, got: %v", err)
	}
}

func TestValidateCountdownCreatePayload_VariantValidation(t *testing.T) {
	base := func(variant string) string {
		return `{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","variant":"` + variant + `"}`
	}
	// Valid standard variant.
	if err := validateCountdownCreatePayload(json.RawMessage(base("standard"))); err != nil {
		t.Fatalf("expected standard variant valid, got: %v", err)
	}
	// Empty variant normalizes to standard.
	if err := validateCountdownCreatePayload(json.RawMessage(`{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase"}`)); err != nil {
		t.Fatalf("expected empty variant valid, got: %v", err)
	}
	// Unknown variant rejected.
	if err := validateCountdownCreatePayload(json.RawMessage(base("chaos"))); err == nil {
		t.Fatal("expected unknown variant rejection")
	}
	// Dynamic without trigger_event_type rejected.
	if err := validateCountdownCreatePayload(json.RawMessage(base("dynamic"))); err == nil {
		t.Fatal("expected dynamic variant without trigger_event_type rejection")
	}
	// Linked without linked_countdown_id rejected.
	if err := validateCountdownCreatePayload(json.RawMessage(base("linked"))); err == nil {
		t.Fatal("expected linked variant without linked_countdown_id rejection")
	}
}

func TestValidateGoldUpdatePayload_Branches(t *testing.T) {
	// No-mutation rejected.
	if err := validateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":3,"handfuls_after":3,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":0}`)); err == nil {
		t.Fatal("expected no-mutation rejection")
	}
	// Handfuls out of range.
	if err := validateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":0,"handfuls_after":10,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":0}`)); err == nil {
		t.Fatal("expected handfuls range error")
	}
	// Bags out of range.
	if err := validateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":0,"handfuls_after":0,"bags_before":0,"bags_after":10,"chests_before":0,"chests_after":0}`)); err == nil {
		t.Fatal("expected bags range error")
	}
	// Chests out of range.
	if err := validateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":0,"handfuls_after":0,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":2}`)); err == nil {
		t.Fatal("expected chests range error")
	}
	// Valid.
	if err := validateGoldUpdatePayload(json.RawMessage(`{"character_id":"char-1","handfuls_before":0,"handfuls_after":3,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":0}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateEquipmentSwapPayload_Branches(t *testing.T) {
	// Invalid item_type.
	if err := validateEquipmentSwapPayload(json.RawMessage(`{"character_id":"char-1","item_id":"s1","item_type":"shield","from":"inventory","to":"active"}`)); err == nil {
		t.Fatal("expected invalid item_type error")
	}
	// Invalid slot.
	if err := validateEquipmentSwapPayload(json.RawMessage(`{"character_id":"char-1","item_id":"s1","item_type":"weapon","from":"backpack","to":"active"}`)); err == nil {
		t.Fatal("expected invalid slot error")
	}
	// From == to.
	if err := validateEquipmentSwapPayload(json.RawMessage(`{"character_id":"char-1","item_id":"s1","item_type":"weapon","from":"active","to":"active"}`)); err == nil {
		t.Fatal("expected from==to error")
	}
	// Valid.
	if err := validateEquipmentSwapPayload(json.RawMessage(`{"character_id":"char-1","item_id":"s1","item_type":"armor","from":"inventory","to":"active"}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateConsumablePayloads_Branches(t *testing.T) {
	if err := validateConsumableUsePayload(json.RawMessage(`{"character_id":" ","consumable_id":"p1","quantity_before":2,"quantity_after":1}`)); err == nil {
		t.Fatal("expected use missing character_id error")
	}
	if err := validateConsumableUsePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":" ","quantity_before":2,"quantity_after":1}`)); err == nil {
		t.Fatal("expected use missing consumable_id error")
	}
	// Use: quantity_before must be positive.
	if err := validateConsumableUsePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":0,"quantity_after":-1}`)); err == nil {
		t.Fatal("expected use quantity_before positive error")
	}
	// Use: quantity_after must be quantity_before - 1.
	if err := validateConsumableUsePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":2,"quantity_after":0}`)); err == nil {
		t.Fatal("expected use quantity_after mismatch error")
	}
	// Use: valid.
	if err := validateConsumableUsePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":2,"quantity_after":1}`)); err != nil {
		t.Fatalf("expected valid use payload, got: %v", err)
	}
	// Acquire: missing character_id / consumable_id.
	if err := validateConsumableAcquirePayload(json.RawMessage(`{"character_id":" ","consumable_id":"p1","quantity_before":1,"quantity_after":2}`)); err == nil {
		t.Fatal("expected acquire missing character_id error")
	}
	if err := validateConsumableAcquirePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":" ","quantity_before":1,"quantity_after":2}`)); err == nil {
		t.Fatal("expected acquire missing consumable_id error")
	}
	// Acquire: quantity_after out of range.
	if err := validateConsumableAcquirePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":5,"quantity_after":6}`)); err == nil {
		t.Fatal("expected acquire quantity_after out of range error")
	}
	// Acquire: quantity_after != quantity_before + 1.
	if err := validateConsumableAcquirePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":1,"quantity_after":3}`)); err == nil {
		t.Fatal("expected acquire quantity_after mismatch error")
	}
	// Acquire: valid.
	if err := validateConsumableAcquirePayload(json.RawMessage(`{"character_id":"char-1","consumable_id":"p1","quantity_before":1,"quantity_after":2}`)); err != nil {
		t.Fatalf("expected valid acquire payload, got: %v", err)
	}
}

func TestValidateLevelUpApplyPayload_Branches(t *testing.T) {
	// Missing advancements.
	if err := validateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[]}`)); err == nil {
		t.Fatal("expected empty advancements error")
	}
	// level_after != level_before + 1.
	if err := validateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":3,"advancements":[{"type":"add_hp_slots"}]}`)); err == nil {
		t.Fatal("expected level_after != level_before+1 error")
	}
	// Valid.
	if err := validateLevelUpApplyPayload(json.RawMessage(`{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}]}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateAdversaryCreateUpdatePayload_Branches(t *testing.T) {
	if err := validateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":" ","name":"Goblin"}`)); err == nil {
		t.Fatal("expected missing adversary_id error")
	}
	if err := validateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":" ","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`)); err == nil {
		t.Fatal("expected missing adversary_entry_id error")
	}
	if err := validateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","name":" "}`)); err == nil {
		t.Fatal("expected missing name error")
	}
	if err := validateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":"entry-1","name":"Goblin","session_id":" ","scene_id":"scene-1"}`)); err == nil {
		t.Fatal("expected missing session_id error")
	}
	if err := validateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":"entry-1","name":"Goblin","session_id":"sess-1","scene_id":" "}`)); err == nil {
		t.Fatal("expected missing scene_id error")
	}
	if err := validateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":"entry-1","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`)); err != nil {
		t.Fatalf("expected valid create payload, got: %v", err)
	}
	if err := validateAdversaryUpdatePayload(json.RawMessage(`{"adversary_id":"adv-1","name":" "}`)); err == nil {
		t.Fatal("expected update missing name error")
	}
	if err := validateAdversaryUpdatePayload(json.RawMessage(`{"adversary_id":"adv-1","adversary_entry_id":"entry-1","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`)); err != nil {
		t.Fatalf("expected valid update payload, got: %v", err)
	}
	if err := validateAdversaryDeletePayload(json.RawMessage(`{"adversary_id":" "}`)); err == nil {
		t.Fatal("expected delete missing adversary_id error")
	}
	if err := validateAdversaryDeletePayload(json.RawMessage(`{"adversary_id":"adv-1"}`)); err != nil {
		t.Fatalf("expected valid delete payload, got: %v", err)
	}
}
