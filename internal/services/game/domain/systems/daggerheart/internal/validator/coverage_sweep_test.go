package validator

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

type rawValidatorCase struct {
	name     string
	raw      string
	validate func([]byte) error
}

func validatorStandardConditionJSON(code string) string {
	label := code
	switch code {
	case "hidden":
		label = "Hidden"
	case "restrained":
		label = "Restrained"
	case "vulnerable":
		label = "Vulnerable"
	case "cloaked":
		label = "Cloaked"
	}
	return `{"id":"` + code + `","class":"standard","standard":"` + code + `","code":"` + code + `","label":"` + label + `"}`
}

func validatorStandardConditionJSONArray(codes ...string) string {
	if len(codes) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(codes))
	for _, code := range codes {
		parts = append(parts, validatorStandardConditionJSON(code))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func TestValidatorSweepForRegisteredPayloadFamilies(t *testing.T) {
	t.Parallel()

	cases := []rawValidatorCase{
		{name: "gm fear set", validate: func(raw []byte) error { return ValidateGMFearSetPayload(raw) }, raw: `{"after":2}`},
		{name: "gm fear changed", validate: func(raw []byte) error { return ValidateGMFearChangedPayload(raw) }, raw: `{"after":2}`},
		{name: "gm move apply", validate: func(raw []byte) error { return ValidateGMMoveApplyPayload(raw) }, raw: `{"target":{"type":"direct_move","kind":"interrupt_and_move","shape":"reveal_danger"},"fear_spent":1}`},
		{name: "gm move applied", validate: func(raw []byte) error { return ValidateGMMoveAppliedPayload(raw) }, raw: `{"target":{"type":"direct_move","kind":"interrupt_and_move","shape":"reveal_danger"},"fear_spent":1}`},
		{name: "character profile replace", validate: func(raw []byte) error { return ValidateCharacterProfileReplacePayload(raw) }, raw: `{"character_id":"char-1","profile":{"class_id":"class.guardian","level":1,"hp_max":6,"stress_max":6,"evasion":10,"major_threshold":1,"severe_threshold":2,"proficiency":1,"armor_score":0,"armor_max":0}}`},
		{name: "character profile replaced", validate: func(raw []byte) error { return ValidateCharacterProfileReplacedPayload(raw) }, raw: `{"character_id":"char-1","profile":{"class_id":"class.guardian","level":1,"hp_max":6,"stress_max":6,"evasion":10,"major_threshold":1,"severe_threshold":2,"proficiency":1,"armor_score":0,"armor_max":0}}`},
		{name: "character state patched", validate: func(raw []byte) error { return ValidateCharacterStatePatchedPayload(raw) }, raw: `{"character_id":"char-1","hp_after":5}`},
		{name: "condition change", validate: func(raw []byte) error { return ValidateConditionChangePayload(raw) }, raw: `{"character_id":"char-1","conditions_after":` + validatorStandardConditionJSONArray("vulnerable") + `}`},
		{name: "condition changed", validate: func(raw []byte) error { return ValidateConditionChangedPayload(raw) }, raw: `{"character_id":"char-1","conditions_after":` + validatorStandardConditionJSONArray("vulnerable") + `}`},
		{name: "adversary condition change", validate: func(raw []byte) error { return ValidateAdversaryConditionChangePayload(raw) }, raw: `{"adversary_id":"adv-1","conditions_after":` + validatorStandardConditionJSONArray("hidden") + `}`},
		{name: "adversary condition changed", validate: func(raw []byte) error { return ValidateAdversaryConditionChangedPayload(raw) }, raw: `{"adversary_id":"adv-1","conditions_after":` + validatorStandardConditionJSONArray("hidden") + `}`},
		{name: "loadout swap", validate: func(raw []byte) error { return ValidateLoadoutSwapPayload(raw) }, raw: `{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`},
		{name: "loadout swapped", validate: func(raw []byte) error { return ValidateLoadoutSwappedPayload(raw) }, raw: `{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`},
		{name: "rest take", validate: func(raw []byte) error { return ValidateRestTakePayload(raw) }, raw: `{"rest_type":"short","gm_fear_before":1,"gm_fear_after":2,"short_rests_before":0,"short_rests_after":1,"refresh_rest":true,"participants":["char-1"]}`},
		{name: "rest taken", validate: func(raw []byte) error { return ValidateRestTakenPayload(raw) }, raw: `{"rest_type":"short","gm_fear_after":2,"short_rests_after":1,"refresh_rest":true,"participants":["char-1"]}`},
		{name: "scene countdown created", validate: func(raw []byte) error { return ValidateSceneCountdownCreatedPayload(raw) }, raw: `{"session_id":"sess-1","scene_id":"scene-1","countdown_id":"cd-1","name":"Doom","tone":"progress","advancement_policy":"action_dynamic","starting_value":4,"remaining_value":4,"loop_behavior":"reset","status":"active"}`},
		{name: "campaign countdown create", validate: func(raw []byte) error { return ValidateCampaignCountdownCreatePayload(raw) }, raw: `{"countdown_id":"camp-cd-1","name":"Long Project","tone":"progress","advancement_policy":"long_rest","starting_value":6,"remaining_value":6,"loop_behavior":"none","status":"active"}`},
		{name: "campaign countdown created", validate: func(raw []byte) error { return ValidateCampaignCountdownCreatedPayload(raw) }, raw: `{"countdown_id":"camp-cd-1","name":"Long Project","tone":"progress","advancement_policy":"long_rest","starting_value":6,"remaining_value":6,"loop_behavior":"none","status":"active"}`},
		{name: "scene countdown advance", validate: func(raw []byte) error { return ValidateSceneCountdownAdvancePayload(raw) }, raw: `{"countdown_id":"cd-1","before_remaining":2,"after_remaining":1,"advanced_by":1,"status_before":"active","status_after":"active"}`},
		{name: "scene countdown advanced", validate: func(raw []byte) error { return ValidateSceneCountdownAdvancedPayload(raw) }, raw: `{"countdown_id":"cd-1","before_remaining":2,"after_remaining":1,"advanced_by":1,"status_before":"active","status_after":"active"}`},
		{name: "campaign countdown advanced", validate: func(raw []byte) error { return ValidateCampaignCountdownAdvancedPayload(raw) }, raw: `{"countdown_id":"camp-cd-1","before_remaining":6,"after_remaining":5,"advanced_by":1,"status_before":"active","status_after":"active"}`},
		{name: "scene countdown trigger resolve", validate: func(raw []byte) error { return ValidateSceneCountdownTriggerResolvePayload(raw) }, raw: `{"countdown_id":"cd-1","starting_value_before":4,"starting_value_after":4,"remaining_value_before":0,"remaining_value_after":4,"status_before":"trigger_pending","status_after":"active"}`},
		{name: "scene countdown trigger resolved", validate: func(raw []byte) error { return ValidateSceneCountdownTriggerResolvedPayload(raw) }, raw: `{"countdown_id":"cd-1","starting_value_before":4,"starting_value_after":4,"remaining_value_before":0,"remaining_value_after":4,"status_before":"trigger_pending","status_after":"active"}`},
		{name: "campaign countdown trigger resolve", validate: func(raw []byte) error { return ValidateCampaignCountdownTriggerResolvePayload(raw) }, raw: `{"countdown_id":"camp-cd-1","starting_value_before":6,"starting_value_after":6,"remaining_value_before":0,"remaining_value_after":6,"status_before":"trigger_pending","status_after":"active"}`},
		{name: "campaign countdown trigger resolved", validate: func(raw []byte) error { return ValidateCampaignCountdownTriggerResolvedPayload(raw) }, raw: `{"countdown_id":"camp-cd-1","starting_value_before":6,"starting_value_after":6,"remaining_value_before":0,"remaining_value_after":6,"status_before":"trigger_pending","status_after":"active"}`},
		{name: "scene countdown delete", validate: func(raw []byte) error { return ValidateSceneCountdownDeletePayload(raw) }, raw: `{"countdown_id":"cd-1"}`},
		{name: "scene countdown deleted", validate: func(raw []byte) error { return ValidateSceneCountdownDeletedPayload(raw) }, raw: `{"countdown_id":"cd-1"}`},
		{name: "campaign countdown delete", validate: func(raw []byte) error { return ValidateCampaignCountdownDeletePayload(raw) }, raw: `{"countdown_id":"camp-cd-1"}`},
		{name: "campaign countdown deleted", validate: func(raw []byte) error { return ValidateCampaignCountdownDeletedPayload(raw) }, raw: `{"countdown_id":"camp-cd-1"}`},
		{name: "damage apply", validate: func(raw []byte) error { return ValidateDamageApplyPayload(raw) }, raw: `{"character_id":"char-1","hp_before":6,"hp_after":3}`},
		{name: "adversary damage apply", validate: func(raw []byte) error { return ValidateAdversaryDamageApplyPayload(raw) }, raw: `{"adversary_id":"adv-1","hp_before":8,"hp_after":3}`},
		{name: "adversary damage applied", validate: func(raw []byte) error { return ValidateAdversaryDamageAppliedPayload(raw) }, raw: `{"adversary_id":"adv-1","hp_after":3}`},
		{name: "downtime move applied", validate: func(raw []byte) error { return ValidateDowntimeMoveAppliedPayload(raw) }, raw: `{"actor_character_id":"char-1","target_character_id":"char-1","move":"clear_all_stress","stress_after":2}`},
		{name: "temporary armor apply", validate: func(raw []byte) error { return ValidateCharacterTemporaryArmorApplyPayload(raw) }, raw: `{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":2,"source_id":"temp-1"}`},
		{name: "temporary armor applied", validate: func(raw []byte) error { return ValidateCharacterTemporaryArmorAppliedPayload(raw) }, raw: `{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":2,"source_id":"temp-1"}`},
		{name: "adversary create", validate: func(raw []byte) error { return ValidateAdversaryCreatePayload(raw) }, raw: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`},
		{name: "adversary created", validate: func(raw []byte) error { return ValidateAdversaryCreatedPayload(raw) }, raw: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`},
		{name: "adversary update", validate: func(raw []byte) error { return ValidateAdversaryUpdatePayload(raw) }, raw: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`},
		{name: "adversary updated", validate: func(raw []byte) error { return ValidateAdversaryUpdatedPayload(raw) }, raw: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`},
		{name: "adversary feature apply", validate: func(raw []byte) error { return ValidateAdversaryFeatureApplyPayload(raw) }, raw: `{"actor_adversary_id":"adv-1","adversary_id":"adv-1","feature_id":"feature.cloaked","feature_states_after":[{"feature_id":"feature.cloaked","status":"active"}]}`},
		{name: "adversary delete", validate: func(raw []byte) error { return ValidateAdversaryDeletePayload(raw) }, raw: `{"adversary_id":"adv-1"}`},
		{name: "adversary deleted", validate: func(raw []byte) error { return ValidateAdversaryDeletedPayload(raw) }, raw: `{"adversary_id":"adv-1"}`},
		{name: "environment create", validate: func(raw []byte) error { return ValidateEnvironmentEntityCreatePayload(raw) }, raw: `{"environment_entity_id":"env-1","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":15,"session_id":"sess-1","scene_id":"scene-1"}`},
		{name: "environment created", validate: func(raw []byte) error { return ValidateEnvironmentEntityCreatedPayload(raw) }, raw: `{"environment_entity_id":"env-1","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":15,"session_id":"sess-1","scene_id":"scene-1"}`},
		{name: "level up applied", validate: func(raw []byte) error { return ValidateLevelUpAppliedPayload(raw) }, raw: `{"character_id":"char-1","level_after":2,"advancements":[{"type":"add_hp_slots"},{"type":"add_stress_slots"}]}`},
		{name: "gold update", validate: func(raw []byte) error { return ValidateGoldUpdatePayload(raw) }, raw: `{"character_id":"char-1","handfuls_before":0,"handfuls_after":3,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":0}`},
		{name: "gold updated", validate: func(raw []byte) error { return ValidateGoldUpdatedPayload(raw) }, raw: `{"character_id":"char-1","handfuls_after":3,"bags_after":0,"chests_after":0}`},
		{name: "domain card acquire", validate: func(raw []byte) error { return ValidateDomainCardAcquirePayload(raw) }, raw: `{"character_id":"char-1","card_id":"card-1","card_level":1,"destination":"vault"}`},
		{name: "domain card acquired", validate: func(raw []byte) error { return ValidateDomainCardAcquiredPayload(raw) }, raw: `{"character_id":"char-1","card_id":"card-1","card_level":1,"destination":"vault"}`},
		{name: "equipment swap", validate: func(raw []byte) error { return ValidateEquipmentSwapPayload(raw) }, raw: `{"character_id":"char-1","item_id":"sword-1","item_type":"weapon","from":"inventory","to":"active"}`},
		{name: "equipment swapped", validate: func(raw []byte) error { return ValidateEquipmentSwappedPayload(raw) }, raw: `{"character_id":"char-1","item_id":"sword-1","item_type":"weapon","from":"inventory","to":"active"}`},
		{name: "consumable use", validate: func(raw []byte) error { return ValidateConsumableUsePayload(raw) }, raw: `{"character_id":"char-1","consumable_id":"potion-1","quantity_before":2,"quantity_after":1}`},
		{name: "consumable used", validate: func(raw []byte) error { return ValidateConsumableUsedPayload(raw) }, raw: `{"character_id":"char-1","consumable_id":"potion-1","quantity_after":1}`},
		{name: "consumable acquire", validate: func(raw []byte) error { return ValidateConsumableAcquirePayload(raw) }, raw: `{"character_id":"char-1","consumable_id":"potion-1","quantity_before":1,"quantity_after":2}`},
		{name: "consumable acquired", validate: func(raw []byte) error { return ValidateConsumableAcquiredPayload(raw) }, raw: `{"character_id":"char-1","consumable_id":"potion-1","quantity_after":2}`},
		{name: "stat modifier change", validate: func(raw []byte) error { return ValidateStatModifierChangePayload(raw) }, raw: `{"character_id":"char-1","modifiers_after":[{"id":"mod-1","target":"evasion","delta":2}]}`},
		{name: "stat modifier changed", validate: func(raw []byte) error { return ValidateStatModifierChangedPayload(raw) }, raw: `{"character_id":"char-1","modifiers_after":[{"id":"mod-1","target":"evasion","delta":2}]}`},
		{name: "class feature apply", validate: func(raw []byte) error { return ValidateClassFeatureApplyPayload(raw) }, raw: `{"actor_character_id":"char-1","feature":"frontline_tank","targets":[{"character_id":"char-1","hope_before":3,"hope_after":0,"armor_before":1,"armor_after":3}]}`},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertErrorContains(t, tc.validate([]byte(tc.raw)), "")
		})
	}
}

func TestValidatorHelpersAdditionalCoverage(t *testing.T) {
	t.Parallel()

	if got := Abs(-3); got != 3 {
		t.Fatalf("Abs(-3) = %d, want 3", got)
	}
	if err := RequireRange(2, 1, 3, "count"); err != nil {
		t.Fatalf("RequireRange() error = %v", err)
	}
	if !IsTemporaryArmorDuration("scene") {
		t.Fatal("IsTemporaryArmorDuration(scene) = false, want true")
	}
	if IsTemporaryArmorDuration("forever") {
		t.Fatal("IsTemporaryArmorDuration(forever) = true, want false")
	}
	if !EqualAdversaryFeatureStates([]rules.AdversaryFeatureState{{FeatureID: "f1", Status: "active"}}, []rules.AdversaryFeatureState{{FeatureID: "f1", Status: "active"}}) {
		t.Fatal("EqualAdversaryFeatureStates(equal) = false, want true")
	}
	if !EqualAdversaryPendingExperience(&rules.AdversaryPendingExperience{Name: "xp", Modifier: 1}, &rules.AdversaryPendingExperience{Name: "xp", Modifier: 1}) {
		t.Fatal("EqualAdversaryPendingExperience(equal) = false, want true")
	}
	if err := ValidateGMMoveTarget(payload.GMMoveTarget{Type: rules.GMMoveTargetTypeDirectMove, Kind: "interrupt_and_move", Shape: "reveal_danger"}); err != nil {
		t.Fatalf("ValidateGMMoveTarget() error = %v", err)
	}
	if !HasRestTakeMutation(payload.RestTakePayload{GMFearBefore: 1, GMFearAfter: 2}) {
		t.Fatal("HasRestTakeMutation() = false, want true")
	}
	if err := ValidateDowntimeMoveAppliedPayloadFields(payload.DowntimeMoveAppliedPayload{
		ActorCharacterID:  "char-1",
		TargetCharacterID: "char-1",
		Move:              "prepare",
		Hope:              intPtr(1),
	}); err != nil {
		t.Fatalf("ValidateDowntimeMoveAppliedPayloadFields() error = %v", err)
	}
}

func intPtr(value int) *int {
	return &value
}
