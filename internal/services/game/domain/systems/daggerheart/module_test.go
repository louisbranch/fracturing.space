package daggerheart

import (
	"errors"
	"testing"
	"time"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	domainmodule "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

type commandValidationCase struct {
	typ            command.Type
	validPayload   string
	invalidPayload string
	actorType      command.ActorType
	actorID        string
}

func commandValidationCases() []commandValidationCase {
	return []commandValidationCase{
		{typ: daggerheartdecider.CommandTypeGMMoveApply, validPayload: `{"target":{"type":"direct_move","kind":"interrupt_and_move","shape":"reveal_danger"},"fear_spent":1}`, invalidPayload: `{"target":{"type":"","kind":"interrupt_and_move","shape":"reveal_danger"},"fear_spent":1}`},
		{typ: daggerheartdecider.CommandTypeGMFearSet, validPayload: `{"after":2}`, invalidPayload: `{"after":"nope"}`, actorType: command.ActorTypeGM, actorID: "gm-1"},
		{typ: daggerheartdecider.CommandTypeCharacterProfileReplace, validPayload: `{"character_id":"char-1","profile":{"class_id":"class.guardian","level":1,"hp_max":6,"stress_max":6,"evasion":10,"major_threshold":1,"severe_threshold":2,"proficiency":1,"armor_score":0,"armor_max":0}}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartdecider.CommandTypeCharacterProfileDelete, validPayload: `{"character_id":"char-1"}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartdecider.CommandTypeHopeSpend, validPayload: `{"character_id":"char-1","amount":1,"before":2,"after":1}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartdecider.CommandTypeStressSpend, validPayload: `{"character_id":"char-1","amount":1,"before":3,"after":2}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartdecider.CommandTypeCharacterStatePatch, validPayload: `{"character_id":"char-1","hp_after":5}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartdecider.CommandTypeConditionChange, validPayload: `{"character_id":"char-1","conditions_after":["vulnerable"]}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartdecider.CommandTypeLoadoutSwap, validPayload: `{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartdecider.CommandTypeRestTake, validPayload: `{"rest_type":"short","gm_fear_before":1,"gm_fear_after":2,"short_rests_before":0,"short_rests_after":1,"refresh_rest":true,"participants":["char-1"]}`, invalidPayload: `{"rest_type":1}`},
		{typ: daggerheartdecider.CommandTypeCountdownCreate, validPayload: `{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`, invalidPayload: `{"countdown_id":1}`},
		{typ: daggerheartdecider.CommandTypeCountdownUpdate, validPayload: `{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false}`, invalidPayload: `{"countdown_id":1}`},
		{typ: daggerheartdecider.CommandTypeCountdownDelete, validPayload: `{"countdown_id":"cd-1"}`, invalidPayload: `{"countdown_id":1}`},
		{typ: daggerheartdecider.CommandTypeDamageApply, validPayload: `{"character_id":"char-1","hp_before":6,"hp_after":3}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartdecider.CommandTypeAdversaryDamageApply, validPayload: `{"adversary_id":"adv-1","hp_before":8,"hp_after":3}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartdecider.CommandTypeCharacterTemporaryArmorApply, validPayload: `{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":2,"source_id":"temp-1"}`, invalidPayload: `{"duration":"short_rest","amount":2}`},
		{typ: daggerheartdecider.CommandTypeAdversaryConditionChange, validPayload: `{"adversary_id":"adv-1","conditions_after":["hidden"]}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartdecider.CommandTypeAdversaryCreate, validPayload: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartdecider.CommandTypeAdversaryUpdate, validPayload: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartdecider.CommandTypeAdversaryFeatureApply, validPayload: `{"actor_adversary_id":"adv-1","adversary_id":"adv-1","feature_id":"feature.cloaked","feature_states_after":[{"feature_id":"feature.cloaked","status":"active"}]}`, invalidPayload: `{"actor_adversary_id":"adv-1","adversary_id":"adv-1","feature_id":"feature.cloaked"}`},
		{typ: daggerheartdecider.CommandTypeAdversaryDelete, validPayload: `{"adversary_id":"adv-1"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartdecider.CommandTypeEnvironmentEntityCreate, validPayload: `{"environment_entity_id":"env-1","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":15,"session_id":"sess-1","scene_id":"scene-1"}`, invalidPayload: `{"environment_entity_id":"env-1","environment_id":"","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":15,"session_id":"sess-1","scene_id":"scene-1"}`},
		{typ: daggerheartdecider.CommandTypeEnvironmentEntityUpdate, validPayload: `{"environment_entity_id":"env-1","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":16,"session_id":"sess-1","scene_id":"scene-2","notes":"moved"}`, invalidPayload: `{"environment_entity_id":"","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":16,"session_id":"sess-1","scene_id":"scene-2"}`},
		{typ: daggerheartdecider.CommandTypeEnvironmentEntityDelete, validPayload: `{"environment_entity_id":"env-1","reason":"cleanup"}`, invalidPayload: `{"environment_entity_id":""}`},
		{typ: daggerheartdecider.CommandTypeMultiTargetDamageApply, validPayload: `{"targets":[{"character_id":"char-1","hp_before":6,"hp_after":3}]}`, invalidPayload: `{"targets":[]}`},
		{typ: daggerheartdecider.CommandTypeLevelUpApply, validPayload: `{"character_id":"char-1","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"},{"type":"add_stress_slots"}]}`, invalidPayload: `{"character_id":"","level_before":1,"level_after":2,"advancements":[{"type":"add_hp_slots"}]}`},
		{typ: daggerheartdecider.CommandTypeClassFeatureApply, validPayload: `{"actor_character_id":"char-1","feature":"frontline_tank","targets":[{"character_id":"char-1","hope_before":3,"hope_after":0,"armor_before":1,"armor_after":3}]}`, invalidPayload: `{"actor_character_id":"char-1","feature":"","targets":[{"character_id":"char-1"}]}`},
		{typ: daggerheartdecider.CommandTypeSubclassFeatureApply, validPayload: `{"actor_character_id":"char-1","feature":"battle_ritual","targets":[{"character_id":"char-1","hope_before":1,"hope_after":3,"stress_before":2,"stress_after":0,"subclass_state_after":{"battle_ritual_used_this_long_rest":true}}]}`, invalidPayload: `{"actor_character_id":"char-1","feature":"","targets":[{"character_id":"char-1"}]}`},
		{typ: daggerheartdecider.CommandTypeBeastformTransform, validPayload: `{"actor_character_id":"char-1","character_id":"char-1","beastform_id":"beastform.wolf","stress_before":1,"stress_after":2,"class_state_after":{"active_beastform":{"beastform_id":"beastform.wolf","base_trait":"agility","attack_trait":"agility","trait_bonus":1,"evasion_bonus":1,"attack_range":"melee","damage_dice":[{"count":1,"sides":8}],"damage_bonus":1,"damage_type":"physical"}}}`, invalidPayload: `{"actor_character_id":"char-1","character_id":"","beastform_id":"beastform.wolf"}`},
		{typ: daggerheartdecider.CommandTypeBeastformDrop, validPayload: `{"actor_character_id":"char-1","character_id":"char-1","beastform_id":"beastform.wolf","source":"beastform.drop","class_state_before":{"active_beastform":{"beastform_id":"beastform.wolf","base_trait":"agility","attack_trait":"agility","attack_range":"melee","damage_dice":[{"count":1,"sides":8}]}},"class_state_after":{}}`, invalidPayload: `{"actor_character_id":"char-1","character_id":"","beastform_id":"beastform.wolf"}`},
		{typ: daggerheartdecider.CommandTypeCompanionExperienceBegin, validPayload: `{"actor_character_id":"char-1","character_id":"char-1","experience_id":"companion-experience.scout","companion_state_before":{"status":"present"},"companion_state_after":{"status":"away","active_experience_id":"companion-experience.scout"}}`, invalidPayload: `{"actor_character_id":"char-1","character_id":"char-1","experience_id":"","companion_state_before":{"status":"present"},"companion_state_after":{"status":"away","active_experience_id":"companion-experience.scout"}}`},
		{typ: daggerheartdecider.CommandTypeCompanionReturn, validPayload: `{"actor_character_id":"char-1","character_id":"char-1","resolution":"experience_completed","stress_before":1,"stress_after":0,"companion_state_before":{"status":"away","active_experience_id":"companion-experience.scout"},"companion_state_after":{"status":"present"}}`, invalidPayload: `{"actor_character_id":"char-1","character_id":"char-1","resolution":"","companion_state_before":{"status":"away","active_experience_id":"companion-experience.scout"},"companion_state_after":{"status":"present"}}`},
		{typ: daggerheartdecider.CommandTypeGoldUpdate, validPayload: `{"character_id":"char-1","handfuls_before":0,"handfuls_after":3,"bags_before":0,"bags_after":0,"chests_before":0,"chests_after":0}`, invalidPayload: `{"character_id":""}`},
		{typ: daggerheartdecider.CommandTypeDomainCardAcquire, validPayload: `{"character_id":"char-1","card_id":"card-1","card_level":1,"destination":"vault"}`, invalidPayload: `{"character_id":"char-1","card_id":"card-1","destination":"backpack"}`},
		{typ: daggerheartdecider.CommandTypeEquipmentSwap, validPayload: `{"character_id":"char-1","item_id":"sword-1","item_type":"weapon","from":"inventory","to":"active"}`, invalidPayload: `{"character_id":"char-1","item_id":"sword-1","item_type":"weapon","from":"active","to":"active"}`},
		{typ: daggerheartdecider.CommandTypeConsumableUse, validPayload: `{"character_id":"char-1","consumable_id":"potion-1","quantity_before":2,"quantity_after":1}`, invalidPayload: `{"character_id":"char-1","consumable_id":"potion-1","quantity_before":0,"quantity_after":-1}`},
		{typ: daggerheartdecider.CommandTypeConsumableAcquire, validPayload: `{"character_id":"char-1","consumable_id":"potion-1","quantity_before":1,"quantity_after":2}`, invalidPayload: `{"character_id":"char-1","consumable_id":"potion-1","quantity_before":5,"quantity_after":6}`},
		{typ: daggerheartdecider.CommandTypeStatModifierChange, validPayload: `{"character_id":"char-1","modifiers_after":[{"id":"mod-1","target":"evasion","delta":2}]}`, invalidPayload: `{"character_id":1}`},
	}
}

type eventValidationCase struct {
	typ            event.Type
	validPayload   string
	invalidPayload string
	actorType      event.ActorType
	actorID        string
}

func eventValidationCases() []eventValidationCase {
	return []eventValidationCase{
		{typ: daggerheartpayload.EventTypeGMMoveApplied, validPayload: `{"target":{"type":"direct_move","kind":"interrupt_and_move","shape":"reveal_danger"},"fear_spent":1}`, invalidPayload: `{"target":{"type":"","kind":"interrupt_and_move","shape":"reveal_danger"},"fear_spent":1}`},
		{typ: daggerheartpayload.EventTypeGMFearChanged, validPayload: `{"after":2}`, invalidPayload: `{"after":"nope"}`, actorType: event.ActorTypeGM, actorID: "gm-1"},
		{typ: daggerheartpayload.EventTypeCharacterProfileReplaced, validPayload: `{"character_id":"char-1","profile":{"class_id":"class.guardian","level":1,"hp_max":6,"stress_max":6,"evasion":10,"major_threshold":1,"severe_threshold":2,"proficiency":1,"armor_score":0,"armor_max":0}}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartpayload.EventTypeCharacterProfileDeleted, validPayload: `{"character_id":"char-1"}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartpayload.EventTypeCharacterStatePatched, validPayload: `{"character_id":"char-1","hp_after":5}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartpayload.EventTypeConditionChanged, validPayload: `{"character_id":"char-1","conditions_after":["vulnerable"]}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartpayload.EventTypeLoadoutSwapped, validPayload: `{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartpayload.EventTypeRestTaken, validPayload: `{"rest_type":"short","gm_fear_after":2,"short_rests_after":1,"refresh_rest":true,"participants":["char-1"]}`, invalidPayload: `{"rest_type":1}`},
		{typ: daggerheartpayload.EventTypeCountdownCreated, validPayload: `{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`, invalidPayload: `{"countdown_id":1}`},
		{typ: daggerheartpayload.EventTypeCountdownUpdated, validPayload: `{"countdown_id":"cd-1","after":3,"delta":1,"looped":false}`, invalidPayload: `{"countdown_id":1}`},
		{typ: daggerheartpayload.EventTypeCountdownDeleted, validPayload: `{"countdown_id":"cd-1"}`, invalidPayload: `{"countdown_id":1}`},
		{typ: daggerheartpayload.EventTypeDamageApplied, validPayload: `{"character_id":"char-1","hp_before":6,"hp_after":3}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartpayload.EventTypeAdversaryDamageApplied, validPayload: `{"adversary_id":"adv-1","hp_before":8,"hp_after":3}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartpayload.EventTypeDowntimeMoveApplied, validPayload: `{"actor_character_id":"char-1","target_character_id":"char-1","move":"clear_all_stress","stress_after":2}`, invalidPayload: `{"actor_character_id":"char-1","move":1}`},
		{typ: daggerheartpayload.EventTypeCharacterTemporaryArmorApplied, validPayload: `{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":2,"source_id":"temp-1"}`, invalidPayload: `{"character_id":1}`},
		{typ: daggerheartpayload.EventTypeBeastformTransformed, validPayload: `{"character_id":"char-1","beastform_id":"beastform.wolf","stress_after":2,"active_beastform":{"beastform_id":"beastform.wolf","base_trait":"agility","attack_trait":"agility","trait_bonus":1,"evasion_bonus":1,"attack_range":"melee","damage_dice":[{"count":1,"sides":8}],"damage_bonus":1,"damage_type":"physical"}}`, invalidPayload: `{"character_id":"char-1","beastform_id":""}`},
		{typ: daggerheartpayload.EventTypeBeastformDropped, validPayload: `{"character_id":"char-1","beastform_id":"beastform.wolf","source":"beastform.drop"}`, invalidPayload: `{"character_id":"char-1","beastform_id":""}`},
		{typ: daggerheartpayload.EventTypeCompanionExperienceBegun, validPayload: `{"character_id":"char-1","experience_id":"companion-experience.scout","companion_state":{"status":"away","active_experience_id":"companion-experience.scout"},"source":"companion.experience.begin"}`, invalidPayload: `{"character_id":"char-1","experience_id":"","companion_state":{"status":"away","active_experience_id":"companion-experience.scout"}}`},
		{typ: daggerheartpayload.EventTypeCompanionReturned, validPayload: `{"character_id":"char-1","resolution":"experience_completed","stress_after":0,"companion_state":{"status":"present"},"source":"companion.return"}`, invalidPayload: `{"character_id":"char-1","resolution":"","companion_state":{"status":"present"}}`},
		{typ: daggerheartpayload.EventTypeAdversaryConditionChanged, validPayload: `{"adversary_id":"adv-1","conditions_after":["hidden"]}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartpayload.EventTypeAdversaryCreated, validPayload: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartpayload.EventTypeAdversaryUpdated, validPayload: `{"adversary_id":"adv-1","adversary_entry_id":"adversary.goblin","name":"Goblin","session_id":"sess-1","scene_id":"scene-1"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartpayload.EventTypeAdversaryDeleted, validPayload: `{"adversary_id":"adv-1"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: daggerheartpayload.EventTypeEnvironmentEntityCreated, validPayload: `{"environment_entity_id":"env-1","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":15,"session_id":"sess-1","scene_id":"scene-1"}`, invalidPayload: `{"environment_entity_id":"env-1","environment_id":"","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":15,"session_id":"sess-1","scene_id":"scene-1"}`},
		{typ: daggerheartpayload.EventTypeEnvironmentEntityUpdated, validPayload: `{"environment_entity_id":"env-1","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":16,"session_id":"sess-1","scene_id":"scene-2","notes":"moved"}`, invalidPayload: `{"environment_entity_id":"","environment_id":"environment.falling-ruins","name":"Falling Ruins","type":"hazard","tier":2,"difficulty":16,"session_id":"sess-1","scene_id":"scene-2"}`},
		{typ: daggerheartpayload.EventTypeEnvironmentEntityDeleted, validPayload: `{"environment_entity_id":"env-1","reason":"cleanup"}`, invalidPayload: `{"environment_entity_id":""}`},
		{typ: daggerheartpayload.EventTypeLevelUpApplied, validPayload: `{"character_id":"char-1","level_after":2,"advancements":[{"type":"add_hp_slots"},{"type":"add_stress_slots"}]}`, invalidPayload: `{"character_id":"","level_after":2,"advancements":[{"type":"add_hp_slots"}]}`},
		{typ: daggerheartpayload.EventTypeGoldUpdated, validPayload: `{"character_id":"char-1","handfuls_after":3,"bags_after":0,"chests_after":0}`, invalidPayload: `{"character_id":""}`},
		{typ: daggerheartpayload.EventTypeDomainCardAcquired, validPayload: `{"character_id":"char-1","card_id":"card-1","card_level":1,"destination":"vault"}`, invalidPayload: `{"character_id":"char-1","card_id":"card-1","destination":"backpack"}`},
		{typ: daggerheartpayload.EventTypeEquipmentSwapped, validPayload: `{"character_id":"char-1","item_id":"sword-1","item_type":"weapon","from":"inventory","to":"active"}`, invalidPayload: `{"character_id":"char-1","item_id":"sword-1","item_type":"weapon","from":"active","to":"active"}`},
		{typ: daggerheartpayload.EventTypeConsumableUsed, validPayload: `{"character_id":"char-1","consumable_id":"potion-1","quantity_after":1}`, invalidPayload: `{"character_id":"char-1","consumable_id":""}`},
		{typ: daggerheartpayload.EventTypeConsumableAcquired, validPayload: `{"character_id":"char-1","consumable_id":"potion-1","quantity_after":2}`, invalidPayload: `{"character_id":"char-1","consumable_id":"potion-1","quantity_after":6}`},
		{typ: daggerheartpayload.EventTypeStatModifierChanged, validPayload: `{"character_id":"char-1","modifiers_after":[{"id":"mod-1","target":"evasion","delta":2}]}`, invalidPayload: `{"character_id":1}`},
	}
}

func TestFoldHandledTypes_DerivedFromRouter(t *testing.T) {
	folder := NewFolder()
	foldTypes := folder.FoldHandledTypes()

	// FoldHandledTypes must match the router's registrations — not the event
	// definitions. This ensures a missing HandleFold registration is caught by
	// startup validators instead of silently passing validation.
	routerTypes := folder.Router.FoldHandledTypes()
	if len(foldTypes) != len(routerTypes) {
		t.Fatalf("FoldHandledTypes() len = %d, router len = %d", len(foldTypes), len(routerTypes))
	}
	routerSet := make(map[event.Type]struct{}, len(routerTypes))
	for _, rt := range routerTypes {
		routerSet[rt] = struct{}{}
	}
	for _, ft := range foldTypes {
		if _, ok := routerSet[ft]; !ok {
			t.Errorf("FoldHandledTypes() contains %s which is not in router registrations", ft)
		}
	}
}

func TestAdapterHandledTypes_DerivedFromRouter(t *testing.T) {
	adapter := NewAdapter(nil)
	handledTypes := adapter.HandledTypes()

	// HandledTypes must match the router's registrations — not the event
	// definitions. This ensures a missing HandleAdapter registration is caught
	// by startup validators instead of silently passing validation.
	routerTypes := adapter.Router.HandledTypes()
	if len(handledTypes) != len(routerTypes) {
		t.Fatalf("HandledTypes() len = %d, router len = %d", len(handledTypes), len(routerTypes))
	}
	routerSet := make(map[event.Type]struct{}, len(routerTypes))
	for _, rt := range routerTypes {
		routerSet[rt] = struct{}{}
	}
	for _, ht := range handledTypes {
		if _, ok := routerSet[ht]; !ok {
			t.Errorf("HandledTypes() contains %s which is not in router registrations", ht)
		}
	}
}

func TestDeciderHandledCommands_DerivedFromCommandDefinitions(t *testing.T) {
	decider := daggerheartdecider.NewDecider(commandTypesFromDefinitions())
	handled := decider.DeciderHandledCommands()

	expected := make(map[command.Type]struct{})
	for _, def := range daggerheartCommandDefinitions {
		expected[def.Type] = struct{}{}
	}

	seen := make(map[command.Type]struct{})
	for _, ct := range handled {
		if _, dup := seen[ct]; dup {
			t.Fatalf("daggerheartdecider.DeciderHandledCommands() contains duplicate command type %s", ct)
		}
		seen[ct] = struct{}{}
		if _, ok := expected[ct]; !ok {
			t.Errorf("daggerheartdecider.DeciderHandledCommands() contains %s which is not in daggerheartCommandDefinitions", ct)
		}
	}
	for typ := range expected {
		if _, ok := seen[typ]; !ok {
			t.Errorf("daggerheartdecider.DeciderHandledCommands() is missing command type %s from daggerheartCommandDefinitions", typ)
		}
	}
}

func TestModuleMetadata(t *testing.T) {
	module := NewModule()

	if module.ID() != SystemID {
		t.Fatalf("module id = %q, want %q", module.ID(), SystemID)
	}
	if module.Version() != SystemVersion {
		t.Fatalf("module version = %q, want %q", module.Version(), SystemVersion)
	}
	if module.Decider() == nil {
		t.Fatal("expected decider")
	}
	if module.Folder() == nil {
		t.Fatal("expected folder")
	}
	if module.StateFactory() == nil {
		t.Fatal("expected state factory")
	}
}

func TestModule_ImplementsCharacterReadinessChecker(t *testing.T) {
	systemModule := NewModule()
	checker, ok := any(systemModule).(domainmodule.CharacterReadinessChecker)
	if !ok {
		t.Fatal("expected daggerheart module to implement CharacterReadinessChecker")
	}

	ready, _ := checker.CharacterReady(
		daggerheartstate.SnapshotState{
			CharacterProfiles: map[ids.CharacterID]daggerheartstate.CharacterProfile{
				"char-1": {ClassID: "class.guardian"},
			},
		},
		character.State{CharacterID: "char-1"},
	)
	if ready {
		t.Fatal("character readiness = true, want false for incomplete workflow")
	}

	ready, reason := checker.CharacterReady(
		daggerheartstate.SnapshotState{
			CharacterProfiles: map[ids.CharacterID]daggerheartstate.CharacterProfile{
				"char-1": {
					ClassID:    "class.guardian",
					SubclassID: "subclass.stalwart",
					Heritage: daggerheartstate.CharacterHeritage{
						FirstFeatureAncestryID:  "heritage.clank",
						FirstFeatureID:          "heritage.clank.feature-1",
						SecondFeatureAncestryID: "heritage.clank",
						SecondFeatureID:         "heritage.clank.feature-2",
						CommunityID:             "heritage.farmer",
					},
					TraitsAssigned:       true,
					Agility:              2,
					Strength:             1,
					Finesse:              1,
					Instinct:             0,
					Presence:             0,
					Knowledge:            -1,
					DetailsRecorded:      true,
					Level:                1,
					HpMax:                6,
					StressMax:            6,
					Evasion:              10,
					StartingWeaponIDs:    []string{"weapon.longsword"},
					StartingArmorID:      "armor.gambeson-armor",
					StartingPotionItemID: StartingPotionMinorHealthID,
					Background:           "Former watch captain",
					Experiences: []daggerheartstate.CharacterProfileExperience{
						{Name: "Shield tactics", Modifier: 2},
						{Name: "Patrol routes", Modifier: 2},
					},
					DomainCardIDs: []string{"domain-card.ward", "domain-card.blade-strike"},
					Connections:   "Trusted by the town guard",
				},
			},
		},
		character.State{CharacterID: "char-1"},
	)
	if !ready {
		t.Fatal("character readiness = false, want true")
	}
	if reason != "" {
		t.Fatalf("character readiness reason = %q, want empty", reason)
	}
}

func TestModuleRegisterCommands_RequiresRegistry(t *testing.T) {
	if err := NewModule().RegisterCommands(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestModuleRegisterEvents_RequiresRegistry(t *testing.T) {
	if err := NewModule().RegisterEvents(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestModuleRegisterCommands_RegistersSysPrefixedOnly(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	definitions := registry.ListDefinitions()
	if got, want := len(definitions), len(daggerheartCommandDefinitions); got != want {
		t.Fatalf("registered command definitions = %d, want %d", got, want)
	}

	canonicalType := command.Type("sys." + SystemID + ".gm_fear.set")
	_, err := registry.ValidateForDecision(command.Command{
		CampaignID:    "camp-1",
		Type:          canonicalType,
		ActorType:     command.ActorTypeGM,
		ActorID:       "gm-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":2}`),
	})
	if err != nil {
		t.Fatalf("canonical command rejected: %v", err)
	}

	_, err = registry.ValidateForDecision(command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.action.gm_fear.set"),
		ActorType:     command.ActorTypeGM,
		ActorID:       "gm-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":2}`),
	})
	if !errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("legacy action command should be unknown, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersSysPrefixedOnly(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	definitions := registry.ListDefinitions()
	if got, want := len(definitions), len(daggerheartEventDefinitions); got != want {
		t.Fatalf("registered event definitions = %d, want %d", got, want)
	}

	canonicalType := event.Type("sys." + SystemID + ".gm_fear_changed")
	_, err := registry.ValidateForAppend(event.Event{
		CampaignID:    "camp-1",
		Type:          canonicalType,
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeGM,
		ActorID:       "gm-1",
		EntityType:    "campaign",
		EntityID:      "camp-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":2}`),
	})
	if err != nil {
		t.Fatalf("canonical event rejected: %v", err)
	}

	_, err = registry.ValidateForAppend(event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.daggerheart.action.gm_fear_changed"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeGM,
		ActorID:       "gm-1",
		EntityType:    "campaign",
		EntityID:      "camp-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":2}`),
	})
	if !errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("legacy action event should be unknown, got %v", err)
	}
}

func TestModuleRegisterEvents_ResolvedNotificationEventsRemoved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	removed := []event.Type{
		event.Type("sys.daggerheart.attack_resolved"),
		event.Type("sys.daggerheart.reaction_resolved"),
		event.Type("sys.daggerheart.adversary_roll_resolved"),
		event.Type("sys.daggerheart.adversary_attack_resolved"),
		event.Type("sys.daggerheart.damage_roll_resolved"),
		event.Type("sys.daggerheart.group_action_resolved"),
		event.Type("sys.daggerheart.tag_team_resolved"),
	}

	definitions := make(map[event.Type]event.Definition, len(removed))
	for _, def := range registry.ListDefinitions() {
		definitions[def.Type] = def
	}

	for _, target := range removed {
		_, ok := definitions[target]
		if ok {
			t.Fatalf("resolved/notification event should not be registered: %s", target)
		}
	}

	for _, def := range registry.ListDefinitions() {
		switch def.Type {
		case daggerheartpayload.EventTypeGMMoveApplied:
			if def.Intent != event.IntentAuditOnly {
				t.Fatalf("event %s intent = %s, want %s", def.Type, def.Intent, event.IntentAuditOnly)
			}
		default:
			if def.Intent != event.IntentProjectionAndReplay {
				t.Fatalf("event %s intent = %s, want %s", def.Type, def.Intent, event.IntentProjectionAndReplay)
			}
		}
	}
}

func TestModuleRegisterCommands_ValidatesAllRegisteredCommands(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	cases := commandValidationCases()
	if len(cases) != len(daggerheartCommandDefinitions) {
		t.Fatalf("command cases = %d, definitions = %d", len(cases), len(daggerheartCommandDefinitions))
	}
	covered := make(map[command.Type]struct{}, len(cases))
	for _, tc := range cases {
		covered[tc.typ] = struct{}{}
	}
	for _, def := range daggerheartCommandDefinitions {
		if _, ok := covered[def.Type]; !ok {
			t.Fatalf("missing command test case for %s", def.Type)
		}
	}

	for _, tc := range cases {
		t.Run(string(tc.typ), func(t *testing.T) {
			actorType := tc.actorType
			if actorType == "" {
				actorType = command.ActorTypeSystem
			}
			valid := command.Command{
				CampaignID:    "camp-1",
				Type:          tc.typ,
				ActorType:     actorType,
				ActorID:       tc.actorID,
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   []byte(tc.validPayload),
			}
			if _, err := registry.ValidateForDecision(valid); err != nil {
				t.Fatalf("valid command rejected: %v", err)
			}

			invalid := valid
			invalid.PayloadJSON = []byte(tc.invalidPayload)
			_, err := registry.ValidateForDecision(invalid)
			if err == nil {
				t.Fatal("expected error")
			}
			if errors.Is(err, command.ErrTypeUnknown) {
				t.Fatalf("expected payload validation error, got %v", err)
			}
		})
	}
}

func TestModuleRegisterEvents_ValidatesAllRegisteredEvents(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	cases := eventValidationCases()
	if len(cases) != len(daggerheartEventDefinitions) {
		t.Fatalf("event cases = %d, definitions = %d", len(cases), len(daggerheartEventDefinitions))
	}
	covered := make(map[event.Type]struct{}, len(cases))
	for _, tc := range cases {
		covered[tc.typ] = struct{}{}
	}
	for _, def := range daggerheartEventDefinitions {
		if _, ok := covered[def.Type]; !ok {
			t.Fatalf("missing event test case for %s", def.Type)
		}
	}

	for _, tc := range cases {
		t.Run(string(tc.typ), func(t *testing.T) {
			actorType := tc.actorType
			if actorType == "" {
				actorType = event.ActorTypeSystem
			}
			valid := event.Event{
				CampaignID:    "camp-1",
				Type:          tc.typ,
				Timestamp:     time.Unix(0, 0).UTC(),
				ActorType:     actorType,
				ActorID:       tc.actorID,
				EntityType:    "action",
				EntityID:      "entity-1",
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   []byte(tc.validPayload),
			}
			if _, err := registry.ValidateForAppend(valid); err != nil {
				t.Fatalf("valid event rejected: %v", err)
			}

			invalid := valid
			invalid.PayloadJSON = []byte(tc.invalidPayload)
			_, err := registry.ValidateForAppend(invalid)
			if err == nil {
				t.Fatal("expected error")
			}
			if errors.Is(err, event.ErrTypeUnknown) {
				t.Fatalf("expected payload validation error, got %v", err)
			}
		})
	}
}

func TestModuleRegisterCommands_RejectsNoOpMutatingPayloads(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	tests := []struct {
		name    string
		typ     command.Type
		payload string
	}{
		{
			name:    "character_state.patch requires changes",
			typ:     daggerheartdecider.CommandTypeCharacterStatePatch,
			payload: `{"character_id":"char-1","hp_before":2,"hp_after":2}`,
		},
		{
			name:    "hope.spend requires non-zero amount",
			typ:     daggerheartdecider.CommandTypeHopeSpend,
			payload: `{"character_id":"char-1","amount":0,"before":2,"after":2}`,
		},
		{
			name:    "hope.spend requires before and after to differ",
			typ:     daggerheartdecider.CommandTypeHopeSpend,
			payload: `{"character_id":"char-1","amount":1,"before":2,"after":2}`,
		},
		{
			name:    "hope.spend requires before-after delta to match amount",
			typ:     daggerheartdecider.CommandTypeHopeSpend,
			payload: `{"character_id":"char-1","amount":2,"before":2,"after":1}`,
		},
		{
			name:    "stress.spend requires non-zero amount",
			typ:     daggerheartdecider.CommandTypeStressSpend,
			payload: `{"character_id":"char-1","amount":0,"before":3,"after":3}`,
		},
		{
			name:    "stress.spend requires before and after to differ",
			typ:     daggerheartdecider.CommandTypeStressSpend,
			payload: `{"character_id":"char-1","amount":1,"before":3,"after":3}`,
		},
		{
			name:    "stress.spend requires before-after delta to match amount",
			typ:     daggerheartdecider.CommandTypeStressSpend,
			payload: `{"character_id":"char-1","amount":2,"before":3,"after":2}`,
		},
		{
			name:    "condition.change requires a change",
			typ:     daggerheartdecider.CommandTypeConditionChange,
			payload: `{"character_id":"char-1","conditions_before":["hidden"],"conditions_after":["hidden"]}`,
		},
		{
			name:    "condition.change requires conditions_after",
			typ:     daggerheartdecider.CommandTypeConditionChange,
			payload: `{"character_id":"char-1","conditions_before":["hidden"]}`,
		},
		{
			name:    "condition.change rejects added removed diff mismatch",
			typ:     daggerheartdecider.CommandTypeConditionChange,
			payload: `{"character_id":"char-1","conditions_before":["hidden"],"conditions_after":["vulnerable"],"added":["restrained"],"removed":["hidden"]}`,
		},
		{
			name:    "countdown.update with no value change is rejected",
			typ:     daggerheartdecider.CommandTypeCountdownUpdate,
			payload: `{"countdown_id":"cd-1","before":3,"after":3,"delta":0,"looped":false}`,
		},
		{
			name:    "damage.apply requires hp or armor change",
			typ:     daggerheartdecider.CommandTypeDamageApply,
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":6}`,
		},
		{
			name:    "adversary_damage.apply requires hp or armor change",
			typ:     daggerheartdecider.CommandTypeAdversaryDamageApply,
			payload: `{"adversary_id":"adv-1","hp_before":8,"hp_after":8}`,
		},
		{
			name:    "rest.take requires durable outcome",
			typ:     daggerheartdecider.CommandTypeRestTake,
			payload: `{"rest_type":"short","gm_fear_before":1,"gm_fear_after":1,"short_rests_before":0,"short_rests_after":0,"refresh_rest":false,"refresh_long_rest":false}`,
		},
		{
			name:    "adversary_condition.change requires a change",
			typ:     daggerheartdecider.CommandTypeAdversaryConditionChange,
			payload: `{"adversary_id":"adv-1","conditions_before":["hidden"],"conditions_after":["hidden"]}`,
		},
		{
			name:    "adversary_condition.change rejects added removed diff mismatch",
			typ:     daggerheartdecider.CommandTypeAdversaryConditionChange,
			payload: `{"adversary_id":"adv-1","conditions_before":["hidden"],"conditions_after":["vulnerable"],"added":["vulnerable"],"removed":["restrained"]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := registry.ValidateForDecision(command.Command{
				CampaignID:    "camp-1",
				Type:          tc.typ,
				ActorType:     command.ActorTypeSystem,
				ActorID:       "system-1",
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   []byte(tc.payload),
			})
			if err == nil {
				t.Fatalf("expected validation failure")
			}
			if errors.Is(err, command.ErrTypeUnknown) {
				t.Fatalf("expected payload validation error, got %v", err)
			}
		})
	}
}

func TestModuleRegisterEvents_RejectsNoOpMutatingPayloads(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	tests := []struct {
		name    string
		typ     event.Type
		payload string
	}{
		{
			name:    "character_state_patched requires at least one after field",
			typ:     daggerheartpayload.EventTypeCharacterStatePatched,
			payload: `{"character_id":"char-1"}`,
		},
		{
			name:    "condition_changed requires conditions_after",
			typ:     daggerheartpayload.EventTypeConditionChanged,
			payload: `{"character_id":"char-1"}`,
		},
		{
			name:    "damage_applied requires hp or armor change",
			typ:     daggerheartpayload.EventTypeDamageApplied,
			payload: `{"character_id":"char-1"}`,
		},
		{
			name:    "adversary_damage_applied requires hp or armor change",
			typ:     daggerheartpayload.EventTypeAdversaryDamageApplied,
			payload: `{"adversary_id":"adv-1"}`,
		},
		{
			name:    "downtime_move_applied requires state change",
			typ:     daggerheartpayload.EventTypeDowntimeMoveApplied,
			payload: `{"character_id":"char-1","move":"clear_all_stress"}`,
		},
		{
			name:    "rest_taken rejects gm_fear out of range",
			typ:     daggerheartpayload.EventTypeRestTaken,
			payload: `{"rest_type":"short","gm_fear_after":99}`,
		},
		{
			name:    "adversary_condition_changed requires conditions_after",
			typ:     daggerheartpayload.EventTypeAdversaryConditionChanged,
			payload: `{"adversary_id":"adv-1"}`,
		},
		{
			name:    "gold_updated rejects handfuls out of range",
			typ:     daggerheartpayload.EventTypeGoldUpdated,
			payload: `{"character_id":"char-1","handfuls_after":10}`,
		},
		{
			name:    "gold_updated rejects bags out of range",
			typ:     daggerheartpayload.EventTypeGoldUpdated,
			payload: `{"character_id":"char-1","bags_after":10}`,
		},
		{
			name:    "gold_updated rejects chests out of range",
			typ:     daggerheartpayload.EventTypeGoldUpdated,
			payload: `{"character_id":"char-1","chests_after":2}`,
		},
		{
			name:    "level_up_applied rejects level out of range",
			typ:     daggerheartpayload.EventTypeLevelUpApplied,
			payload: `{"character_id":"char-1","level_after":0,"advancements":[{"type":"add_hp_slots"}]}`,
		},
		{
			name:    "level_up_applied rejects empty advancements",
			typ:     daggerheartpayload.EventTypeLevelUpApplied,
			payload: `{"character_id":"char-1","level_after":2}`,
		},
		{
			name:    "consumable_acquired rejects empty character_id",
			typ:     daggerheartpayload.EventTypeConsumableAcquired,
			payload: `{"character_id":"","consumable_id":"potion-1","quantity_after":2}`,
		},
		{
			name:    "consumable_acquired rejects empty consumable_id",
			typ:     daggerheartpayload.EventTypeConsumableAcquired,
			payload: `{"character_id":"char-1","consumable_id":"","quantity_after":2}`,
		},
		{
			name:    "consumable_used rejects empty character_id",
			typ:     daggerheartpayload.EventTypeConsumableUsed,
			payload: `{"character_id":"","consumable_id":"potion-1"}`,
		},
		{
			name:    "loadout_swapped rejects empty character_id",
			typ:     daggerheartpayload.EventTypeLoadoutSwapped,
			payload: `{"character_id":"","card_id":"card-1"}`,
		},
		{
			name:    "loadout_swapped rejects empty card_id",
			typ:     daggerheartpayload.EventTypeLoadoutSwapped,
			payload: `{"character_id":"char-1","card_id":""}`,
		},
		{
			name:    "condition_changed rejects empty character_id",
			typ:     daggerheartpayload.EventTypeConditionChanged,
			payload: `{"character_id":"","conditions_after":["hidden"]}`,
		},
		{
			name:    "adversary_condition_changed rejects empty adversary_id",
			typ:     daggerheartpayload.EventTypeAdversaryConditionChanged,
			payload: `{"adversary_id":"","conditions_after":["hidden"]}`,
		},
		{
			name:    "rest_taken rejects empty rest_type",
			typ:     daggerheartpayload.EventTypeRestTaken,
			payload: `{"rest_type":"","gm_fear_after":2}`,
		},
		{
			name:    "downtime_move_applied rejects empty move",
			typ:     daggerheartpayload.EventTypeDowntimeMoveApplied,
			payload: `{"character_id":"char-1","move":"","stress_after":2}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := registry.ValidateForAppend(event.Event{
				CampaignID:    "camp-1",
				Type:          tc.typ,
				Timestamp:     time.Unix(0, 0).UTC(),
				ActorType:     event.ActorTypeSystem,
				ActorID:       "system-1",
				EntityType:    "character",
				EntityID:      "entity-1",
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   []byte(tc.payload),
			})
			if err == nil {
				t.Fatalf("expected validation failure")
			}
			if errors.Is(err, event.ErrTypeUnknown) {
				t.Fatalf("expected payload validation error, got %v", err)
			}
		})
	}
}

func TestModuleRegisterCommands_AllowsConditionAddedWithoutBefore(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	tests := []struct {
		name    string
		typ     command.Type
		payload string
	}{
		{
			name:    "character condition change add from empty",
			typ:     daggerheartdecider.CommandTypeConditionChange,
			payload: `{"character_id":"char-1","conditions_after":["hidden"],"added":["hidden"]}`,
		},
		{
			name:    "adversary condition change add from empty",
			typ:     daggerheartdecider.CommandTypeAdversaryConditionChange,
			payload: `{"adversary_id":"adv-1","conditions_after":["hidden"],"added":["hidden"]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := registry.ValidateForDecision(command.Command{
				CampaignID:    "camp-1",
				Type:          tc.typ,
				ActorType:     command.ActorTypeSystem,
				ActorID:       "system-1",
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   []byte(tc.payload),
			})
			if err != nil {
				t.Fatalf("expected payload to be valid, got %v", err)
			}
		})
	}
}

func TestModuleRegisterEvents_AllowsConditionAddedWithoutBefore(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	tests := []struct {
		name    string
		typ     event.Type
		payload string
	}{
		{
			name:    "condition changed add from empty",
			typ:     daggerheartpayload.EventTypeConditionChanged,
			payload: `{"character_id":"char-1","conditions_after":["hidden"],"added":["hidden"]}`,
		},
		{
			name:    "adversary condition changed add from empty",
			typ:     daggerheartpayload.EventTypeAdversaryConditionChanged,
			payload: `{"adversary_id":"adv-1","conditions_after":["hidden"],"added":["hidden"]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := registry.ValidateForAppend(event.Event{
				CampaignID:    "camp-1",
				Type:          tc.typ,
				Timestamp:     time.Unix(0, 0).UTC(),
				ActorType:     event.ActorTypeSystem,
				ActorID:       "system-1",
				EntityType:    "character",
				EntityID:      "entity-1",
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   []byte(tc.payload),
			})
			if err != nil {
				t.Fatalf("expected payload to be valid, got %v", err)
			}
		})
	}
}

func TestModuleRegisterCommands_DamageApplyRejectsAdapterInvalidPayloads(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "marks above cap",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"marks":5}`,
		},
		{
			name:    "armor spent negative",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"armor_spent":-1}`,
		},
		{
			name:    "roll seq must be positive",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"roll_seq":0}`,
		},
		{
			name:    "severity must be known value",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"severity":"extreme"}`,
		},
		{
			name:    "source ids must not contain empty values",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"source_character_ids":["char-2",""]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := registry.ValidateForDecision(command.Command{
				CampaignID:    "camp-1",
				Type:          daggerheartdecider.CommandTypeDamageApply,
				ActorType:     command.ActorTypeSystem,
				ActorID:       "system-1",
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   []byte(tc.payload),
			})
			if err == nil {
				t.Fatalf("expected validation failure")
			}
			if errors.Is(err, command.ErrTypeUnknown) {
				t.Fatalf("expected payload validation error, got %v", err)
			}
		})
	}
}

func TestModuleRegisterEvents_DamageAppliedRejectsAdapterInvalidPayloads(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "marks above cap",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"marks":5}`,
		},
		{
			name:    "armor spent negative",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"armor_spent":-1}`,
		},
		{
			name:    "roll seq must be positive",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"roll_seq":0}`,
		},
		{
			name:    "severity must be known value",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"severity":"extreme"}`,
		},
		{
			name:    "source ids must not contain empty values",
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":5,"source_character_ids":["char-2",""]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := registry.ValidateForAppend(event.Event{
				CampaignID:    "camp-1",
				Type:          daggerheartpayload.EventTypeDamageApplied,
				Timestamp:     time.Unix(0, 0).UTC(),
				ActorType:     event.ActorTypeSystem,
				ActorID:       "system-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   []byte(tc.payload),
			})
			if err == nil {
				t.Fatalf("expected validation failure")
			}
			if errors.Is(err, event.ErrTypeUnknown) {
				t.Fatalf("expected payload validation error, got %v", err)
			}
		})
	}
}
