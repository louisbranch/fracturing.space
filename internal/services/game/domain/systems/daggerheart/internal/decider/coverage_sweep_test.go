package decider

import (
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

type commandValidationCase struct {
	typ          command.Type
	validPayload string
	actorType    command.ActorType
	actorID      string
}

func standardConditionJSON(code string) string {
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

func standardConditionJSONArray(codes ...string) string {
	if len(codes) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(codes))
	for _, code := range codes {
		parts = append(parts, standardConditionJSON(code))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func commandValidationCases() []commandValidationCase {
	return []commandValidationCase{
		{typ: CommandTypeGMMoveApply, validPayload: `{"target":{"type":"direct_move","kind":"interrupt_and_move","shape":"reveal_danger"},"fear_spent":1}`},
		{typ: CommandTypeGMFearSet, validPayload: `{"after":2}`, actorType: command.ActorTypeGM, actorID: "gm-1"},
		{typ: CommandTypeCharacterProfileReplace, validPayload: `{"character_id":"char-1","profile":{"class_id":"class.guardian","level":1,"hp_max":6,"stress_max":6,"evasion":10,"major_threshold":1,"severe_threshold":2,"proficiency":1,"armor_score":0,"armor_max":0}}`},
		{typ: CommandTypeCharacterProfileDelete, validPayload: `{"character_id":"char-1"}`},
		{typ: CommandTypeHopeSpend, validPayload: `{"character_id":"char-1","amount":1,"before":2,"after":1}`},
		{typ: CommandTypeStressSpend, validPayload: `{"character_id":"char-1","amount":1,"before":3,"after":2}`},
		{typ: CommandTypeCharacterStatePatch, validPayload: `{"character_id":"char-1","hp_after":5}`},
		{typ: CommandTypeConditionChange, validPayload: `{"character_id":"char-1","conditions_after":` + standardConditionJSONArray("vulnerable") + `}`},
		{typ: CommandTypeLoadoutSwap, validPayload: `{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`},
		{typ: CommandTypeRestTake, validPayload: `{"rest_type":"short","gm_fear_before":1,"gm_fear_after":2,"short_rests_before":0,"short_rests_after":1,"refresh_rest":true,"participants":["char-1"]}`},
		{typ: CommandTypeSceneCountdownCreate, validPayload: `{"session_id":"sess-1","scene_id":"scene-1","countdown_id":"cd-1","name":"Doom","tone":"progress","advancement_policy":"action_dynamic","starting_value":4,"remaining_value":4,"loop_behavior":"reset","status":"active"}`},
		{typ: CommandTypeSceneCountdownAdvance, validPayload: `{"countdown_id":"cd-1","before_remaining":2,"after_remaining":1,"advanced_by":1,"status_before":"active","status_after":"active"}`},
		{typ: CommandTypeSceneCountdownTriggerResolve, validPayload: `{"countdown_id":"cd-1","starting_value_before":4,"starting_value_after":4,"remaining_value_before":0,"remaining_value_after":4,"status_before":"trigger_pending","status_after":"active"}`},
		{typ: CommandTypeSceneCountdownDelete, validPayload: `{"countdown_id":"cd-1"}`},
		{typ: CommandTypeCampaignCountdownCreate, validPayload: `{"countdown_id":"camp-cd-1","name":"Long Project","tone":"progress","advancement_policy":"long_rest","starting_value":6,"remaining_value":6,"loop_behavior":"none","status":"active"}`},
		{typ: CommandTypeCampaignCountdownAdvance, validPayload: `{"countdown_id":"camp-cd-1","before_remaining":6,"after_remaining":5,"advanced_by":1,"status_before":"active","status_after":"active"}`},
		{typ: CommandTypeCampaignCountdownTriggerResolve, validPayload: `{"countdown_id":"camp-cd-1","starting_value_before":6,"starting_value_after":6,"remaining_value_before":0,"remaining_value_after":6,"status_before":"trigger_pending","status_after":"active"}`},
		{typ: CommandTypeCampaignCountdownDelete, validPayload: `{"countdown_id":"camp-cd-1"}`},
		{typ: CommandTypeDamageApply, validPayload: `{"character_id":"char-1","hp_before":6,"hp_after":3}`},
		{typ: CommandTypeAdversaryDamageApply, validPayload: `{"adversary_id":"adv-1","hp_before":8,"hp_after":3}`},
		{typ: CommandTypeCharacterTemporaryArmorApply, validPayload: `{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":2,"source_id":"temp-1"}`},
		{typ: CommandTypeAdversaryConditionChange, validPayload: `{"adversary_id":"adv-1","conditions_after":` + standardConditionJSONArray("hidden") + `}`},
		{typ: CommandTypeAdversaryCreate, validPayload: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`},
		{typ: CommandTypeAdversaryUpdate, validPayload: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`},
		{typ: CommandTypeAdversaryFeatureApply, validPayload: `{"actor_adversary_id":"adv-1","adversary_id":"adv-1","feature_id":"feature.cloaked","feature_states_after":[{"feature_id":"feature.cloaked","status":"active"}]}`},
		{typ: CommandTypeAdversaryDelete, validPayload: `{"adversary_id":"adv-1"}`},
		{typ: CommandTypeEnvironmentEntityCreate, validPayload: `{"environment_entity_id":"env-1","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":15,"session_id":"sess-1","scene_id":"scene-1"}`},
		{typ: CommandTypeEnvironmentEntityUpdate, validPayload: `{"environment_entity_id":"env-1","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":16,"session_id":"sess-1","scene_id":"scene-2","notes":"moved"}`},
		{typ: CommandTypeEnvironmentEntityDelete, validPayload: `{"environment_entity_id":"env-1","reason":"cleanup"}`},
		{typ: CommandTypeMultiTargetDamageApply, validPayload: `{"targets":[{"character_id":"char-1","hp_before":6,"hp_after":3}]}`},
		{typ: CommandTypeLevelUpApply, validPayload: `{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"},{"type":"add_stress_slots"}]}`},
		{typ: CommandTypeClassFeatureApply, validPayload: `{"actor_character_id":"char-1","feature":"frontline_tank","targets":[{"character_id":"char-1","hope_before":3,"hope_after":0,"armor_before":1,"armor_after":3}]}`},
		{typ: CommandTypeSubclassFeatureApply, validPayload: `{"actor_character_id":"char-1","feature":"battle_ritual","targets":[{"character_id":"char-1","hope_before":1,"hope_after":3,"stress_before":2,"stress_after":0,"subclass_state_after":{"battle_ritual_used_this_long_rest":true}}]}`},
		{typ: CommandTypeBeastformTransform, validPayload: `{"actor_character_id":"char-1","character_id":"char-1","beastform_id":"beastform.wolf","stress_before":1,"stress_after":2,"class_state_after":{"active_beastform":{"beastform_id":"beastform.wolf","base_trait":"agility","attack_trait":"agility","trait_bonus":1,"evasion_bonus":1,"attack_range":"melee","damage_dice":[{"count":1,"sides":8}],"damage_bonus":1,"damage_type":"physical"}}}`},
		{typ: CommandTypeBeastformDrop, validPayload: `{"actor_character_id":"char-1","character_id":"char-1","beastform_id":"beastform.wolf","source":"beastform.drop","class_state_before":{"active_beastform":{"beastform_id":"beastform.wolf","base_trait":"agility","attack_trait":"agility","attack_range":"melee","damage_dice":[{"count":1,"sides":8}]}},"class_state_after":{}}`},
		{typ: CommandTypeCompanionExperienceBegin, validPayload: `{"actor_character_id":"char-1","character_id":"char-1","experience_id":"companion-experience.scout","companion_state_before":{"status":"present"},"companion_state_after":{"status":"away","active_experience_id":"companion-experience.scout"}}`},
		{typ: CommandTypeCompanionReturn, validPayload: `{"actor_character_id":"char-1","character_id":"char-1","resolution":"experience_completed","stress_before":1,"stress_after":0,"companion_state_before":{"status":"away","active_experience_id":"companion-experience.scout"},"companion_state_after":{"status":"present"}}`},
		{typ: CommandTypeGoldUpdate, validPayload: `{"character_id":"char-1","handfuls_before":0,"handfuls_after":3,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":0}`},
		{typ: CommandTypeDomainCardAcquire, validPayload: `{"character_id":"char-1","card_id":"card-1","card_level":1,"destination":"vault"}`},
		{typ: CommandTypeEquipmentSwap, validPayload: `{"character_id":"char-1","item_id":"sword-1","item_type":"weapon","from":"inventory","to":"active"}`},
		{typ: CommandTypeConsumableUse, validPayload: `{"character_id":"char-1","consumable_id":"potion-1","quantity_before":2,"quantity_after":1}`},
		{typ: CommandTypeConsumableAcquire, validPayload: `{"character_id":"char-1","consumable_id":"potion-1","quantity_before":1,"quantity_after":2}`},
		{typ: CommandTypeStatModifierChange, validPayload: `{"character_id":"char-1","modifiers_after":[{"id":"mod-1","target":"evasion","delta":2}]}`},
	}
}

func TestDecisionHandlersReturnNonEmptyOutcomeForValidPayloads(t *testing.T) {
	t.Parallel()

	now := func() time.Time { return time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC) }
	for _, tc := range commandValidationCases() {
		tc := tc
		t.Run(string(tc.typ), func(t *testing.T) {
			t.Parallel()

			actorType := tc.actorType
			if actorType == "" {
				actorType = command.ActorTypeSystem
			}
			decision := NewDecider([]command.Type{tc.typ}).Decide(nil, command.Command{
				CampaignID:    "camp-1",
				Type:          tc.typ,
				ActorType:     actorType,
				ActorID:       tc.actorID,
				SystemID:      "daggerheart",
				SystemVersion: "v1",
				PayloadJSON:   []byte(tc.validPayload),
			}, now)
			if len(decision.Events) == 0 && len(decision.Rejections) == 0 {
				t.Fatalf("%s returned empty decision", tc.typ)
			}
		})
	}
}

func TestDeciderHandledCommandsReturnsConfiguredSurface(t *testing.T) {
	t.Parallel()

	handled := []command.Type{CommandTypeGMFearSet, CommandTypeDamageApply}
	got := NewDecider(handled).DeciderHandledCommands()
	if len(got) != len(handled) {
		t.Fatalf("len(DeciderHandledCommands()) = %d, want %d", len(got), len(handled))
	}
}
