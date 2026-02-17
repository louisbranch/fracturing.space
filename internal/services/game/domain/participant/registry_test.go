package participant

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestRegisterCommands_ValidatesJoinPayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"Alice","role":"PLAYER"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.join"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":1,"name":"Alice","role":"PLAYER"}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesJoinedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.joined"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "p-1",
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"Alice","role":"PLAYER"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"participant_id":1,"name":"Alice","role":"PLAYER"}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_JoinedRequiresEntityTargetAddressing(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	base := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.joined"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","name":"Alice","role":"PLAYER"}`),
	}

	_, err := registry.ValidateForAppend(base)
	if err == nil {
		t.Fatal("expected missing entity type error")
	}
	if !errors.Is(err, event.ErrEntityTypeRequired) {
		t.Fatalf("expected ErrEntityTypeRequired, got %v", err)
	}

	withType := base
	withType.EntityType = "participant"
	_, err = registry.ValidateForAppend(withType)
	if err == nil {
		t.Fatal("expected missing entity id error")
	}
	if !errors.Is(err, event.ErrEntityIDRequired) {
		t.Fatalf("expected ErrEntityIDRequired, got %v", err)
	}

	withTypeAndID := withType
	withTypeAndID.EntityID = "p-1"
	if _, err := registry.ValidateForAppend(withTypeAndID); err != nil {
		t.Fatalf("valid addressed event rejected: %v", err)
	}
}

func TestRegisterCommands_ValidatesUpdatePayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","fields":{"name":"Alice"}}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","fields":{"name":1}}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesUpdatedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.updated"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "p-1",
		PayloadJSON: []byte(`{"participant_id":"p-1","fields":{"name":"Alice"}}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"participant_id":"p-1","fields":{"name":1}}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesLeavePayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.leave"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","reason":"bye"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.leave"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":1}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesLeftPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.left"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "p-1",
		PayloadJSON: []byte(`{"participant_id":"p-1"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"participant_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesBindPayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.bind"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"u-1"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.bind"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":1,"user_id":"u-1"}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesBoundPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.bound"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "p-1",
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"u-1"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"participant_id":1,"user_id":"u-1"}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesUnbindPayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.unbind"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("participant.unbind"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":1}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesUnboundPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("participant.unbound"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "p-1",
		PayloadJSON: []byte(`{"participant_id":"p-1"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"participant_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesSeatReassignPayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("seat.reassign"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"u-2"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("seat.reassign"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"participant_id":1,"user_id":"u-2"}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesSeatReassignedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("seat.reassigned"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    "p-1",
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"u-2"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"participant_id":1,"user_id":"u-2"}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}
