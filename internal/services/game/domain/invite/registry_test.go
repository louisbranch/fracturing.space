package invite

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
		Type:        command.Type("invite.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":1,"participant_id":"p-1"}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesClaimPayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.claim"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","user_id":"user-1","jti":"jwt-1"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.claim"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":1}`),
	}
	_, err := registry.ValidateForDecision(invalidCommand)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, command.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterCommands_ValidatesRevokePayload(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	validCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.revoke"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.revoke"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":1}`),
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
		Type:        command.Type("invite.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","status":"PENDING"}`),
	}
	if _, err := registry.ValidateForDecision(validCommand); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	invalidCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","status":1}`),
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
		Type:        event.Type("invite.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","status":"pending"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"invite_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesClaimedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("invite.claimed"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","user_id":"user-1","jti":"jwt-1"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"invite_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}

func TestRegisterEvents_ValidatesRevokedPayload(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	validEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("invite.revoked"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"invite_id":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
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
		Type:        event.Type("invite.updated"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","status":"pending"}`),
	}
	if _, err := registry.ValidateForAppend(validEvent); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalidEvent := validEvent
	invalidEvent.PayloadJSON = []byte(`{"invite_id":"inv-1","status":1}`)
	_, err := registry.ValidateForAppend(invalidEvent)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, event.ErrTypeUnknown) {
		t.Fatalf("expected payload validation error, got %v", err)
	}
}
