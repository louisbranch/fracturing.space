package daggerheart

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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
		{typ: commandTypeGMFearSet, validPayload: `{"after":2}`, invalidPayload: `{"after":"nope"}`, actorType: command.ActorTypeGM, actorID: "gm-1"},
		{typ: commandTypeHopeSpend, validPayload: `{"character_id":"char-1","amount":1,"before":2,"after":1}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeStressSpend, validPayload: `{"character_id":"char-1","amount":1,"before":3,"after":2}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeCharacterStatePatch, validPayload: `{"character_id":"char-1","hp_after":5}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeConditionChange, validPayload: `{"character_id":"char-1","conditions_after":["vulnerable"]}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeLoadoutSwap, validPayload: `{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeRestTake, validPayload: `{"rest_type":"short","gm_fear_before":1,"gm_fear_after":2,"short_rests_before":0,"short_rests_after":1,"refresh_rest":true}`, invalidPayload: `{"rest_type":1}`},
		{typ: commandTypeCountdownCreate, validPayload: `{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`, invalidPayload: `{"countdown_id":1}`},
		{typ: commandTypeCountdownUpdate, validPayload: `{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false}`, invalidPayload: `{"countdown_id":1}`},
		{typ: commandTypeCountdownDelete, validPayload: `{"countdown_id":"cd-1"}`, invalidPayload: `{"countdown_id":1}`},
		{typ: commandTypeDamageApply, validPayload: `{"character_id":"char-1","hp_before":6,"hp_after":3}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeAdversaryDamageApply, validPayload: `{"adversary_id":"adv-1","hp_before":8,"hp_after":3}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeDowntimeMoveApply, validPayload: `{"character_id":"char-1","move":"clear_all_stress","stress_before":3,"stress_after":2}`, invalidPayload: `{"character_id":"char-1","move":""}`},
		{typ: commandTypeCharacterTemporaryArmorApply, validPayload: `{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":2,"source_id":"temp-1"}`, invalidPayload: `{"duration":"short_rest","amount":2}`},
		{typ: commandTypeAdversaryConditionChange, validPayload: `{"adversary_id":"adv-1","conditions_after":["hidden"]}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeAdversaryCreate, validPayload: `{"adversary_id":"adv-1","name":"Goblin"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeAdversaryUpdate, validPayload: `{"adversary_id":"adv-1","name":"Goblin"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeAdversaryDelete, validPayload: `{"adversary_id":"adv-1"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeMultiTargetDamageApply, validPayload: `{"targets":[{"character_id":"char-1","hp_before":6,"hp_after":3}]}`, invalidPayload: `{"targets":[]}`},
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
		{typ: EventTypeGMFearChanged, validPayload: `{"before":1,"after":2}`, invalidPayload: `{"before":1,"after":"nope"}`, actorType: event.ActorTypeGM, actorID: "gm-1"},
		{typ: EventTypeCharacterStatePatched, validPayload: `{"character_id":"char-1","hp_after":5}`, invalidPayload: `{"character_id":1}`},
		{typ: EventTypeConditionChanged, validPayload: `{"character_id":"char-1","conditions_after":["vulnerable"]}`, invalidPayload: `{"character_id":1}`},
		{typ: EventTypeLoadoutSwapped, validPayload: `{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`, invalidPayload: `{"character_id":1}`},
		{typ: EventTypeRestTaken, validPayload: `{"rest_type":"short","gm_fear_before":1,"gm_fear_after":2,"short_rests_before":0,"short_rests_after":1,"refresh_rest":true}`, invalidPayload: `{"rest_type":1}`},
		{typ: EventTypeCountdownCreated, validPayload: `{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`, invalidPayload: `{"countdown_id":1}`},
		{typ: EventTypeCountdownUpdated, validPayload: `{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false}`, invalidPayload: `{"countdown_id":1}`},
		{typ: EventTypeCountdownDeleted, validPayload: `{"countdown_id":"cd-1"}`, invalidPayload: `{"countdown_id":1}`},
		{typ: EventTypeDamageApplied, validPayload: `{"character_id":"char-1","hp_before":6,"hp_after":3}`, invalidPayload: `{"character_id":1}`},
		{typ: EventTypeAdversaryDamageApplied, validPayload: `{"adversary_id":"adv-1","hp_before":8,"hp_after":3}`, invalidPayload: `{"adversary_id":1}`},
		{typ: EventTypeDowntimeMoveApplied, validPayload: `{"character_id":"char-1","move":"clear_all_stress","stress_before":3,"stress_after":2}`, invalidPayload: `{"character_id":"char-1","move":1}`},
		{typ: EventTypeCharacterTemporaryArmorApplied, validPayload: `{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":2,"source_id":"temp-1"}`, invalidPayload: `{"character_id":1}`},
		{typ: EventTypeAdversaryConditionChanged, validPayload: `{"adversary_id":"adv-1","conditions_after":["hidden"]}`, invalidPayload: `{"adversary_id":1}`},
		{typ: EventTypeAdversaryCreated, validPayload: `{"adversary_id":"adv-1","name":"Goblin"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: EventTypeAdversaryUpdated, validPayload: `{"adversary_id":"adv-1","name":"Goblin"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: EventTypeAdversaryDeleted, validPayload: `{"adversary_id":"adv-1"}`, invalidPayload: `{"adversary_id":1}`},
	}
}

func TestFoldHandledTypes_DerivedFromRouter(t *testing.T) {
	folder := NewFolder()
	foldTypes := folder.FoldHandledTypes()

	// FoldHandledTypes must match the router's registrations — not the event
	// definitions. This ensures a missing HandleFold registration is caught by
	// startup validators instead of silently passing validation.
	routerTypes := folder.router.FoldHandledTypes()
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
	routerTypes := adapter.router.HandledTypes()
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
	decider := Decider{}
	handled := decider.DeciderHandledCommands()

	expected := make(map[command.Type]struct{})
	for _, def := range daggerheartCommandDefinitions {
		expected[def.Type] = struct{}{}
	}

	seen := make(map[command.Type]struct{})
	for _, ct := range handled {
		if _, dup := seen[ct]; dup {
			t.Fatalf("DeciderHandledCommands() contains duplicate command type %s", ct)
		}
		seen[ct] = struct{}{}
		if _, ok := expected[ct]; !ok {
			t.Errorf("DeciderHandledCommands() contains %s which is not in daggerheartCommandDefinitions", ct)
		}
	}
	for typ := range expected {
		if _, ok := seen[typ]; !ok {
			t.Errorf("DeciderHandledCommands() is missing command type %s from daggerheartCommandDefinitions", typ)
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

	ready, _ := checker.CharacterReady(map[string]any{"daggerheart": map[string]any{"class_id": "class.guardian"}})
	if ready {
		t.Fatal("character readiness = true, want false for incomplete workflow")
	}

	ready, reason := checker.CharacterReady(map[string]any{
		"daggerheart": map[string]any{
			"class_id":                "class.guardian",
			"subclass_id":             "subclass.stalwart",
			"ancestry_id":             "heritage.clank",
			"community_id":            "heritage.farmer",
			"traits_assigned":         true,
			"agility":                 2,
			"strength":                1,
			"finesse":                 1,
			"instinct":                0,
			"presence":                0,
			"knowledge":               -1,
			"details_recorded":        true,
			"level":                   1,
			"hp_max":                  6,
			"stress_max":              6,
			"evasion":                 10,
			"starting_weapon_ids":     []string{"weapon.longsword"},
			"starting_armor_id":       "armor.gambeson-armor",
			"starting_potion_item_id": StartingPotionMinorHealthID,
			"background":              "Former watch captain",
			"experiences": []map[string]any{
				{"name": "Shield tactics", "modifier": 2},
			},
			"domain_card_ids": []string{"domain-card.ward"},
			"connections":     "Trusted by the town guard",
		},
	})
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
		PayloadJSON:   []byte(`{"before":1,"after":2}`),
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
		PayloadJSON:   []byte(`{"before":1,"after":2}`),
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
		event.Type("sys.daggerheart.gm_move_applied"),
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
		if def.Intent != event.IntentProjectionAndReplay {
			t.Fatalf("event %s intent = %s, want %s", def.Type, def.Intent, event.IntentProjectionAndReplay)
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
			typ:     commandTypeCharacterStatePatch,
			payload: `{"character_id":"char-1","hp_before":2,"hp_after":2}`,
		},
		{
			name:    "hope.spend requires non-zero amount",
			typ:     commandTypeHopeSpend,
			payload: `{"character_id":"char-1","amount":0,"before":2,"after":2}`,
		},
		{
			name:    "hope.spend requires before and after to differ",
			typ:     commandTypeHopeSpend,
			payload: `{"character_id":"char-1","amount":1,"before":2,"after":2}`,
		},
		{
			name:    "hope.spend requires before-after delta to match amount",
			typ:     commandTypeHopeSpend,
			payload: `{"character_id":"char-1","amount":2,"before":2,"after":1}`,
		},
		{
			name:    "stress.spend requires non-zero amount",
			typ:     commandTypeStressSpend,
			payload: `{"character_id":"char-1","amount":0,"before":3,"after":3}`,
		},
		{
			name:    "stress.spend requires before and after to differ",
			typ:     commandTypeStressSpend,
			payload: `{"character_id":"char-1","amount":1,"before":3,"after":3}`,
		},
		{
			name:    "stress.spend requires before-after delta to match amount",
			typ:     commandTypeStressSpend,
			payload: `{"character_id":"char-1","amount":2,"before":3,"after":2}`,
		},
		{
			name:    "condition.change requires a change",
			typ:     commandTypeConditionChange,
			payload: `{"character_id":"char-1","conditions_before":["hidden"],"conditions_after":["hidden"]}`,
		},
		{
			name:    "condition.change requires conditions_after",
			typ:     commandTypeConditionChange,
			payload: `{"character_id":"char-1","conditions_before":["hidden"]}`,
		},
		{
			name:    "condition.change rejects added removed diff mismatch",
			typ:     commandTypeConditionChange,
			payload: `{"character_id":"char-1","conditions_before":["hidden"],"conditions_after":["vulnerable"],"added":["restrained"],"removed":["hidden"]}`,
		},
		{
			name:    "countdown.update with no value change is rejected",
			typ:     commandTypeCountdownUpdate,
			payload: `{"countdown_id":"cd-1","before":3,"after":3,"delta":0,"looped":false}`,
		},
		{
			name:    "damage.apply requires hp or armor change",
			typ:     commandTypeDamageApply,
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":6}`,
		},
		{
			name:    "adversary_damage.apply requires hp or armor change",
			typ:     commandTypeAdversaryDamageApply,
			payload: `{"adversary_id":"adv-1","hp_before":8,"hp_after":8}`,
		},
		{
			name:    "downtime_move.apply requires state change",
			typ:     commandTypeDowntimeMoveApply,
			payload: `{"character_id":"char-1","move":"clear_all_stress","stress_before":2,"stress_after":2}`,
		},
		{
			name:    "rest.take requires rest change or character patches",
			typ:     commandTypeRestTake,
			payload: `{"rest_type":"short","gm_fear_before":1,"gm_fear_after":1,"short_rests_before":0,"short_rests_after":0,"refresh_rest":false,"refresh_long_rest":false}`,
		},
		{
			name:    "adversary_condition.change requires a change",
			typ:     commandTypeAdversaryConditionChange,
			payload: `{"adversary_id":"adv-1","conditions_before":["hidden"],"conditions_after":["hidden"]}`,
		},
		{
			name:    "adversary_condition.change rejects added removed diff mismatch",
			typ:     commandTypeAdversaryConditionChange,
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
			name:    "character_state_patched requires a change",
			typ:     EventTypeCharacterStatePatched,
			payload: `{"character_id":"char-1","hp_before":2,"hp_after":2}`,
		},
		{
			name:    "gm_fear_changed requires a change",
			typ:     EventTypeGMFearChanged,
			payload: `{"before":2,"after":2}`,
		},
		{
			name:    "condition_changed requires a change",
			typ:     EventTypeConditionChanged,
			payload: `{"character_id":"char-1","conditions_before":["hidden"],"conditions_after":["hidden"]}`,
		},
		{
			name:    "condition_changed requires conditions_after",
			typ:     EventTypeConditionChanged,
			payload: `{"character_id":"char-1","conditions_before":["hidden"]}`,
		},
		{
			name:    "condition_changed rejects added removed diff mismatch",
			typ:     EventTypeConditionChanged,
			payload: `{"character_id":"char-1","conditions_before":["hidden"],"conditions_after":["vulnerable"],"added":["restrained"],"removed":["hidden"]}`,
		},
		{
			name:    "countdown_updated requires a change",
			typ:     EventTypeCountdownUpdated,
			payload: `{"countdown_id":"cd-1","before":3,"after":3,"delta":0,"looped":false}`,
		},
		{
			name:    "damage_applied requires hp or armor change",
			typ:     EventTypeDamageApplied,
			payload: `{"character_id":"char-1","hp_before":6,"hp_after":6}`,
		},
		{
			name:    "adversary_damage_applied requires hp or armor change",
			typ:     EventTypeAdversaryDamageApplied,
			payload: `{"adversary_id":"adv-1","hp_before":8,"hp_after":8}`,
		},
		{
			name:    "downtime_move_applied requires state change",
			typ:     EventTypeDowntimeMoveApplied,
			payload: `{"character_id":"char-1","move":"clear_all_stress","stress_before":2,"stress_after":2}`,
		},
		{
			name:    "rest_taken requires rest change or character patches",
			typ:     EventTypeRestTaken,
			payload: `{"rest_type":"short","gm_fear_before":1,"gm_fear_after":1,"short_rests_before":0,"short_rests_after":0,"refresh_rest":false,"refresh_long_rest":false}`,
		},
		{
			name:    "adversary_condition_changed requires a change",
			typ:     EventTypeAdversaryConditionChanged,
			payload: `{"adversary_id":"adv-1","conditions_before":["hidden"],"conditions_after":["hidden"]}`,
		},
		{
			name:    "adversary_condition_changed rejects added removed diff mismatch",
			typ:     EventTypeAdversaryConditionChanged,
			payload: `{"adversary_id":"adv-1","conditions_before":["hidden"],"conditions_after":["vulnerable"],"added":["vulnerable"],"removed":["restrained"]}`,
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
			typ:     commandTypeConditionChange,
			payload: `{"character_id":"char-1","conditions_after":["hidden"],"added":["hidden"]}`,
		},
		{
			name:    "adversary condition change add from empty",
			typ:     commandTypeAdversaryConditionChange,
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
			typ:     EventTypeConditionChanged,
			payload: `{"character_id":"char-1","conditions_after":["hidden"],"added":["hidden"]}`,
		},
		{
			name:    "adversary condition changed add from empty",
			typ:     EventTypeAdversaryConditionChanged,
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
				Type:          commandTypeDamageApply,
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
				Type:          EventTypeDamageApplied,
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
