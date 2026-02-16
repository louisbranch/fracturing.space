package daggerheart

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestModuleRegisterCommands_RegistersGMFearSet(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.gm_fear.set"),
		ActorType:     command.ActorTypeGM,
		ActorID:       "gm-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":2}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"after":"nope"}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersGMFearChanged(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.gm_fear_changed"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeGM,
		ActorID:       "gm-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"before":1,"after":2}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"before":1,"after":"nope"}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersCharacterStatePatch(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.character_state.patch"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","hp_after":5}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersConditionChange(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.condition.change"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","conditions_after":["shaken"]}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersCharacterStatePatched(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.character_state_patched"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","hp_after":5}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersConditionChanged(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.condition_changed"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","conditions_after":["shaken"]}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersHopeSpend(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.hope.spend"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","amount":1,"before":2,"after":1}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersHopeSpent(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.hope_spent"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","amount":1,"before":2,"after":1}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersStressSpend(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.stress.spend"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","amount":1,"before":3,"after":2}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersStressSpent(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.stress_spent"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","amount":1,"before":3,"after":2}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersLoadoutSwap(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.loadout.swap"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersLoadoutSwapped(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.loadout_swapped"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersRestTake(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.rest.take"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"rest_type":"short"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"rest_type":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersRestTaken(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.rest_taken"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"rest_type":"short"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"rest_type":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersDamageApply(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.damage.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersDamageApplied(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.damage_applied"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersAttackResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.attack.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","roll_seq":4,"targets":["char-2"]}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersAttackResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.attack_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","roll_seq":4,"targets":["char-2"]}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersReactionResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.reaction.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","roll_seq":5,"outcome":"success"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersReactionResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.reaction_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","roll_seq":5,"outcome":"success"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersAdversaryAttackResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_attack.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","roll_seq":6,"targets":["char-1"]}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersAdversaryAttackResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.adversary_attack_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","roll_seq":6,"targets":["char-1"]}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersDamageRollResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.damage_roll.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","roll_seq":7}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersDamageRollResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.damage_roll_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","roll_seq":7}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersGroupActionResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.group_action.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"leader_character_id":"char-1","leader_roll_seq":1,"support_successes":1,"support_failures":0,"support_modifier":1}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"leader_character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersGroupActionResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.group_action_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"leader_character_id":"char-1","leader_roll_seq":1,"support_successes":1,"support_failures":0,"support_modifier":1}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"leader_character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersTagTeamResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.tag_team.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"first_character_id":"char-1","first_roll_seq":1,"second_character_id":"char-2","second_roll_seq":2,"selected_character_id":"char-1","selected_roll_seq":1}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"first_character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersCountdownCreate(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.countdown.create"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"countdown_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersCountdownUpdate(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.countdown.update"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"countdown_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersCountdownDelete(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.countdown.delete"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"countdown_id":"cd-1"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"countdown_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersTagTeamResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.tag_team_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"first_character_id":"char-1","first_roll_seq":1,"second_character_id":"char-2","second_roll_seq":2,"selected_character_id":"char-1","selected_roll_seq":1}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"first_character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersCountdownCreated(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.countdown_created"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"countdown_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersCountdownUpdated(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.countdown_updated"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"countdown_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersCountdownDeleted(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.countdown_deleted"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"countdown_id":"cd-1"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"countdown_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersAdversaryActionResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_action.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","roll_seq":1,"difficulty":10,"dramatic":false,"auto_success":true,"success":true}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersAdversaryActionResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.adversary_action_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","roll_seq":1,"difficulty":10,"dramatic":false,"auto_success":true,"success":true}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersAdversaryRollResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_roll.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","roll_seq":1,"rolls":[12],"roll":12,"modifier":2,"total":14}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersAdversaryRollResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.adversary_roll_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","roll_seq":1,"rolls":[12],"roll":12,"modifier":2,"total":14}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersBlazeOfGloryResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.blaze_of_glory.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","life_state_after":"dead"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersBlazeOfGloryResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.blaze_of_glory_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","life_state_after":"dead"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersAdversaryDamageApply(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_damage.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersAdversaryDamageApplied(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.adversary_damage_applied"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersDowntimeMoveApply(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.downtime_move.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","move":"clear_all_stress"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":"char-1","move":""}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersDowntimeMoveApplied(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.downtime_move_applied"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","move":"clear_all_stress"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":"char-1","move":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersDeathMoveResolve(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.death_move.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","move":"avoid_death","life_state_after":"alive"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"character_id":"char-1","move":"avoid_death"}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersDeathMoveResolved(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.death_move_resolved"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","move":"avoid_death","life_state_after":"alive"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":"char-1","move":""}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersGMMoveApply(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.gm_move.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"move":"change_environment","fear_spent":1,"severity":"soft"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"move":""}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersGMMoveApplied(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.gm_move_applied"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"move":"change_environment","fear_spent":1,"severity":"soft"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"move":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersAdversaryConditionChange(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_condition.change"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","conditions_after":["hidden"]}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := validCommand
	invalidCommand.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersAdversaryConditionChanged(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.adversary_condition_changed"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","conditions_after":["hidden"]}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterCommands_RegistersAdversaryCRUD(t *testing.T) {
	registry := command.NewRegistry()
	module := NewModule()
	if err := module.RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCreate := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary.create"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","name":"Goblin"}`),
	}
	if _, err := registry.ValidateForDecision(validCreate); err != nil {
		t.Fatalf("valid create command rejected: %v", err)
	}

	validUpdate := validCreate
	validUpdate.Type = command.Type("action.adversary.update")
	if _, err := registry.ValidateForDecision(validUpdate); err != nil {
		t.Fatalf("valid update command rejected: %v", err)
	}

	validDelete := validCreate
	validDelete.Type = command.Type("action.adversary.delete")
	validDelete.PayloadJSON = []byte(`{"adversary_id":"adv-1"}`)
	if _, err := registry.ValidateForDecision(validDelete); err != nil {
		t.Fatalf("valid delete command rejected: %v", err)
	}

	invalidCommand := validCreate
	invalidCommand.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestModuleRegisterEvents_RegistersAdversaryCRUD(t *testing.T) {
	registry := event.NewRegistry()
	module := NewModule()
	if err := module.RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validCreated := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("action.adversary_created"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     event.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","name":"Goblin"}`),
	}
	if _, err := registry.ValidateForAppend(validCreated); err != nil {
		t.Fatalf("valid created event rejected: %v", err)
	}

	validUpdated := validCreated
	validUpdated.Type = event.Type("action.adversary_updated")
	if _, err := registry.ValidateForAppend(validUpdated); err != nil {
		t.Fatalf("valid updated event rejected: %v", err)
	}

	validDeleted := validCreated
	validDeleted.Type = event.Type("action.adversary_deleted")
	validDeleted.PayloadJSON = []byte(`{"adversary_id":"adv-1"}`)
	if _, err := registry.ValidateForAppend(validDeleted); err != nil {
		t.Fatalf("valid deleted event rejected: %v", err)
	}

	invalidEvent := validCreated
	invalidEvent.PayloadJSON = []byte(`{"adversary_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}
