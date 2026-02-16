package character

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestRegisterCommands_ValidatesCreatePayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"PC"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":1,"name":"Aria","kind":"PC"}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesUpdatePayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"name":"Aria"}}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"name":1}}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesDeletePayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.delete"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.delete"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":1}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesProfileUpdatePayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.profile_update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","system_profile":{"daggerheart":{}}}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("character.profile_update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":1}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesCreatedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("character.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "character",
		EntityID:    "char-1",
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"PC"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":1,"name":"Aria","kind":"PC"}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_CreatedRequiresEntityTargetAddressing(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	base := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("character.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"PC"}`),
	}

	_, err := registry.ValidateForAppend(base)
	if err == nil {
		t.Fatal("expected missing entity type error")
	}
	if !errors.Is(err, event.ErrEntityTypeRequired) {
		t.Fatalf("expected ErrEntityTypeRequired, got %v", err)
	}

	withType := base
	withType.EntityType = "character"
	_, err = registry.ValidateForAppend(withType)
	if err == nil {
		t.Fatal("expected missing entity id error")
	}
	if !errors.Is(err, event.ErrEntityIDRequired) {
		t.Fatalf("expected ErrEntityIDRequired, got %v", err)
	}

	withTypeAndID := withType
	withTypeAndID.EntityID = "char-1"
	if _, err := registry.ValidateForAppend(withTypeAndID); err != nil {
		t.Fatalf("valid addressed event rejected: %v", err)
	}
}

func TestRegisterEvents_ValidatesUpdatedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("character.updated"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "character",
		EntityID:    "char-1",
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"name":"Aria"}}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"character_id":"char-1","fields":{"name":1}}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesDeletedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("character.deleted"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "character",
		EntityID:    "char-1",
		PayloadJSON: []byte(`{"character_id":"char-1"}`),
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

func TestRegisterEvents_ValidatesProfileUpdatedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("character.profile_updated"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "character",
		EntityID:    "char-1",
		PayloadJSON: []byte(`{"character_id":"char-1","system_profile":{"daggerheart":{}}}`),
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
