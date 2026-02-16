package action

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestRegisterCommands_ValidatesActionPayloads(t *testing.T) {
	registry := command.NewRegistry()
	if err := RegisterCommands(registry); err != nil {
		t.Fatalf("register commands: %v", err)
	}

	tests := []struct {
		name    string
		valid   command.Command
		invalid command.Command
	}{
		{
			name: "roll resolve",
			valid: command.Command{
				CampaignID:  "camp-1",
				Type:        command.Type("action.roll.resolve"),
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"request_id":"req-1"}`),
			},
			invalid: command.Command{
				CampaignID:  "camp-1",
				Type:        command.Type("action.roll.resolve"),
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"request_id":1}`),
			},
		},
		{
			name: "outcome apply",
			valid: command.Command{
				CampaignID:  "camp-1",
				Type:        command.Type("action.outcome.apply"),
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"request_id":"req-1"}`),
			},
			invalid: command.Command{
				CampaignID:  "camp-1",
				Type:        command.Type("action.outcome.apply"),
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"request_id":1}`),
			},
		},
		{
			name: "outcome reject",
			valid: command.Command{
				CampaignID:  "camp-1",
				Type:        command.Type("action.outcome.reject"),
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"request_id":"req-1"}`),
			},
			invalid: command.Command{
				CampaignID:  "camp-1",
				Type:        command.Type("action.outcome.reject"),
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"request_id":1}`),
			},
		},
		{
			name: "note add",
			valid: command.Command{
				CampaignID:  "camp-1",
				Type:        command.Type("action.note.add"),
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"content":"note"}`),
			},
			invalid: command.Command{
				CampaignID:  "camp-1",
				Type:        command.Type("action.note.add"),
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{"content":1}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := registry.ValidateForDecision(tt.valid); err != nil {
				t.Fatalf("valid command rejected: %v", err)
			}
			_, err := registry.ValidateForDecision(tt.invalid)
			if err == nil {
				t.Fatal("expected error")
			}
			if errors.Is(err, command.ErrTypeUnknown) {
				t.Fatalf("expected payload validation error, got %v", err)
			}
		})
	}
}

func TestRegisterEvents_ValidatesActionPayloads(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	tests := []struct {
		name    string
		valid   event.Event
		invalid event.Event
	}{
		{
			name: "roll resolved",
			valid: event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "roll",
				EntityID:    "req-1",
				PayloadJSON: []byte(`{"request_id":"req-1"}`),
			},
			invalid: event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "roll",
				EntityID:    "req-1",
				PayloadJSON: []byte(`{"request_id":1}`),
			},
		},
		{
			name: "outcome applied",
			valid: event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "outcome",
				EntityID:    "req-1",
				PayloadJSON: []byte(`{"request_id":"req-1"}`),
			},
			invalid: event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "outcome",
				EntityID:    "req-1",
				PayloadJSON: []byte(`{"request_id":1}`),
			},
		},
		{
			name: "outcome rejected",
			valid: event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_rejected"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "outcome",
				EntityID:    "req-1",
				PayloadJSON: []byte(`{"request_id":"req-1"}`),
			},
			invalid: event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_rejected"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "outcome",
				EntityID:    "req-1",
				PayloadJSON: []byte(`{"request_id":1}`),
			},
		},
		{
			name: "note added",
			valid: event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.note_added"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "actor-1",
				EntityType:  "note",
				EntityID:    "note-1",
				PayloadJSON: []byte(`{"content":"note"}`),
			},
			invalid: event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.note_added"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "actor-1",
				EntityType:  "note",
				EntityID:    "note-1",
				PayloadJSON: []byte(`{"content":1}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := registry.ValidateForAppend(tt.valid); err != nil {
				t.Fatalf("valid event rejected: %v", err)
			}
			_, err := registry.ValidateForAppend(tt.invalid)
			if err == nil {
				t.Fatal("expected error")
			}
			if errors.Is(err, event.ErrTypeUnknown) {
				t.Fatalf("expected payload validation error, got %v", err)
			}
		})
	}
}

func TestRegisterEvents_RollResolvedRequiresEntityTargetAddressing(t *testing.T) {
	registry := event.NewRegistry()
	if err := RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}

	base := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("action.roll_resolved"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: []byte(`{"request_id":"req-1"}`),
	}

	_, err := registry.ValidateForAppend(base)
	if err == nil {
		t.Fatal("expected missing entity type error")
	}
	if !errors.Is(err, event.ErrEntityTypeRequired) {
		t.Fatalf("expected ErrEntityTypeRequired, got %v", err)
	}

	withType := base
	withType.EntityType = "action"
	_, err = registry.ValidateForAppend(withType)
	if err == nil {
		t.Fatal("expected missing entity id error")
	}
	if !errors.Is(err, event.ErrEntityIDRequired) {
		t.Fatalf("expected ErrEntityIDRequired, got %v", err)
	}

	withTypeAndID := withType
	withTypeAndID.EntityID = "req-1"
	if _, err := registry.ValidateForAppend(withTypeAndID); err != nil {
		t.Fatalf("valid addressed event rejected: %v", err)
	}
}
