package daggerheart

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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
		{typ: commandTypeCharacterStatePatch, validPayload: `{"character_id":"char-1","hp_after":5}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeConditionChange, validPayload: `{"character_id":"char-1","conditions_after":["shaken"]}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeHopeSpend, validPayload: `{"character_id":"char-1","amount":1,"before":2,"after":1}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeStressSpend, validPayload: `{"character_id":"char-1","amount":1,"before":3,"after":2}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeLoadoutSwap, validPayload: `{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeRestTake, validPayload: `{"rest_type":"short"}`, invalidPayload: `{"rest_type":1}`},
		{typ: commandTypeAttackResolve, validPayload: `{"character_id":"char-1","roll_seq":4,"targets":["char-2"]}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeReactionResolve, validPayload: `{"character_id":"char-1","roll_seq":5,"outcome":"success"}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeAdversaryRollResolve, validPayload: `{"adversary_id":"adv-1","roll_seq":1,"rolls":[12],"roll":12,"modifier":2,"total":14}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeAdversaryAttackResolve, validPayload: `{"adversary_id":"adv-1","roll_seq":6,"targets":["char-1"]}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeDamageRollResolve, validPayload: `{"character_id":"char-1","roll_seq":7}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeGroupActionResolve, validPayload: `{"leader_character_id":"char-1","leader_roll_seq":1,"support_successes":1,"support_failures":0,"support_modifier":1}`, invalidPayload: `{"leader_character_id":1}`},
		{typ: commandTypeTagTeamResolve, validPayload: `{"first_character_id":"char-1","first_roll_seq":1,"second_character_id":"char-2","second_roll_seq":2,"selected_character_id":"char-1","selected_roll_seq":1}`, invalidPayload: `{"first_character_id":1}`},
		{typ: commandTypeCountdownCreate, validPayload: `{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`, invalidPayload: `{"countdown_id":1}`},
		{typ: commandTypeCountdownUpdate, validPayload: `{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false}`, invalidPayload: `{"countdown_id":1}`},
		{typ: commandTypeCountdownDelete, validPayload: `{"countdown_id":"cd-1"}`, invalidPayload: `{"countdown_id":1}`},
		{typ: commandTypeAdversaryActionResolve, validPayload: `{"adversary_id":"adv-1","roll_seq":1,"difficulty":10,"dramatic":false,"auto_success":true,"success":true}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeDamageApply, validPayload: `{"character_id":"char-1"}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeAdversaryDamageApply, validPayload: `{"adversary_id":"adv-1"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeDowntimeMoveApply, validPayload: `{"character_id":"char-1","move":"clear_all_stress"}`, invalidPayload: `{"character_id":"char-1","move":""}`},
		{typ: commandTypeDeathMoveResolve, validPayload: `{"character_id":"char-1","move":"avoid_death","life_state_after":"alive"}`, invalidPayload: `{"character_id":"char-1","move":"avoid_death"}`},
		{typ: commandTypeBlazeOfGloryResolve, validPayload: `{"character_id":"char-1","life_state_after":"dead"}`, invalidPayload: `{"character_id":1}`},
		{typ: commandTypeGMMoveApply, validPayload: `{"move":"change_environment","fear_spent":1,"severity":"soft"}`, invalidPayload: `{"move":""}`},
		{typ: commandTypeAdversaryConditionChange, validPayload: `{"adversary_id":"adv-1","conditions_after":["hidden"]}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeAdversaryCreate, validPayload: `{"adversary_id":"adv-1","name":"Goblin"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeAdversaryUpdate, validPayload: `{"adversary_id":"adv-1","name":"Goblin"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: commandTypeAdversaryDelete, validPayload: `{"adversary_id":"adv-1"}`, invalidPayload: `{"adversary_id":1}`},
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
		{typ: eventTypeGMFearChanged, validPayload: `{"before":1,"after":2}`, invalidPayload: `{"before":1,"after":"nope"}`, actorType: event.ActorTypeGM, actorID: "gm-1"},
		{typ: eventTypeCharacterStatePatched, validPayload: `{"character_id":"char-1","hp_after":5}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeConditionChanged, validPayload: `{"character_id":"char-1","conditions_after":["shaken"]}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeHopeSpent, validPayload: `{"character_id":"char-1","amount":1,"before":2,"after":1}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeStressSpent, validPayload: `{"character_id":"char-1","amount":1,"before":3,"after":2}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeLoadoutSwapped, validPayload: `{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeRestTaken, validPayload: `{"rest_type":"short"}`, invalidPayload: `{"rest_type":1}`},
		{typ: eventTypeAttackResolved, validPayload: `{"character_id":"char-1","roll_seq":4,"targets":["char-2"]}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeReactionResolved, validPayload: `{"character_id":"char-1","roll_seq":5,"outcome":"success"}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeAdversaryRollResolved, validPayload: `{"adversary_id":"adv-1","roll_seq":1,"rolls":[12],"roll":12,"modifier":2,"total":14}`, invalidPayload: `{"adversary_id":1}`},
		{typ: eventTypeAdversaryAttackResolved, validPayload: `{"adversary_id":"adv-1","roll_seq":6,"targets":["char-1"]}`, invalidPayload: `{"adversary_id":1}`},
		{typ: eventTypeDamageRollResolved, validPayload: `{"character_id":"char-1","roll_seq":7}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeGroupActionResolved, validPayload: `{"leader_character_id":"char-1","leader_roll_seq":1,"support_successes":1,"support_failures":0,"support_modifier":1}`, invalidPayload: `{"leader_character_id":1}`},
		{typ: eventTypeTagTeamResolved, validPayload: `{"first_character_id":"char-1","first_roll_seq":1,"second_character_id":"char-2","second_roll_seq":2,"selected_character_id":"char-1","selected_roll_seq":1}`, invalidPayload: `{"first_character_id":1}`},
		{typ: eventTypeCountdownCreated, validPayload: `{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`, invalidPayload: `{"countdown_id":1}`},
		{typ: eventTypeCountdownUpdated, validPayload: `{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false}`, invalidPayload: `{"countdown_id":1}`},
		{typ: eventTypeCountdownDeleted, validPayload: `{"countdown_id":"cd-1"}`, invalidPayload: `{"countdown_id":1}`},
		{typ: eventTypeAdversaryActionResolved, validPayload: `{"adversary_id":"adv-1","roll_seq":1,"difficulty":10,"dramatic":false,"auto_success":true,"success":true}`, invalidPayload: `{"adversary_id":1}`},
		{typ: eventTypeDamageApplied, validPayload: `{"character_id":"char-1"}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeAdversaryDamageApplied, validPayload: `{"adversary_id":"adv-1"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: eventTypeDowntimeMoveApplied, validPayload: `{"character_id":"char-1","move":"clear_all_stress"}`, invalidPayload: `{"character_id":"char-1","move":1}`},
		{typ: eventTypeDeathMoveResolved, validPayload: `{"character_id":"char-1","move":"avoid_death","life_state_after":"alive"}`, invalidPayload: `{"character_id":"char-1","move":""}`},
		{typ: eventTypeBlazeOfGloryResolved, validPayload: `{"character_id":"char-1","life_state_after":"dead"}`, invalidPayload: `{"character_id":1}`},
		{typ: eventTypeGMMoveApplied, validPayload: `{"move":"change_environment","fear_spent":1,"severity":"soft"}`, invalidPayload: `{"move":1}`},
		{typ: eventTypeAdversaryConditionChanged, validPayload: `{"adversary_id":"adv-1","conditions_after":["hidden"]}`, invalidPayload: `{"adversary_id":1}`},
		{typ: eventTypeAdversaryCreated, validPayload: `{"adversary_id":"adv-1","name":"Goblin"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: eventTypeAdversaryUpdated, validPayload: `{"adversary_id":"adv-1","name":"Goblin"}`, invalidPayload: `{"adversary_id":1}`},
		{typ: eventTypeAdversaryDeleted, validPayload: `{"adversary_id":"adv-1"}`, invalidPayload: `{"adversary_id":1}`},
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
	if module.Projector() == nil {
		t.Fatal("expected projector")
	}
	if module.StateFactory() == nil {
		t.Fatal("expected state factory")
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
