package action

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestDecideActionCommands_EmitsEvents(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name           string
		commandType    string
		payloadJSON    string
		wantType       string
		wantEntityType string
		wantEntityID   string
	}{
		{
			name:           "roll resolved",
			commandType:    "action.roll.resolve",
			payloadJSON:    `{"request_id":"req-1","roll_seq":1,"results":{"d20":20},"outcome":"success"}`,
			wantType:       "action.roll_resolved",
			wantEntityType: "roll",
			wantEntityID:   "req-1",
		},
		{
			name:           "outcome applied",
			commandType:    "action.outcome.apply",
			payloadJSON:    `{"request_id":"req-2","roll_seq":2,"targets":["c1"],"requires_complication":true,"applied_changes":[{"character_id":"c1","field":"hp","before":10,"after":8}]}`,
			wantType:       "action.outcome_applied",
			wantEntityType: "outcome",
			wantEntityID:   "req-2",
		},
		{
			name:           "outcome rejected",
			commandType:    "action.outcome.reject",
			payloadJSON:    `{"request_id":"req-3","roll_seq":3,"reason_code":"INVALID","message":"bad"}`,
			wantType:       "action.outcome_rejected",
			wantEntityType: "outcome",
			wantEntityID:   "req-3",
		},
		{
			name:           "note added",
			commandType:    "story.note.add",
			payloadJSON:    `{"content":"note","character_id":"char-1"}`,
			wantType:       "story.note_added",
			wantEntityType: "note",
			wantEntityID:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Decide(State{}, command.Command{
				CampaignID:  "camp-1",
				Type:        command.Type(tt.commandType),
				ActorType:   command.ActorTypeSystem,
				SessionID:   "sess-1",
				PayloadJSON: []byte(tt.payloadJSON),
			}, func() time.Time { return now })
			if len(decision.Rejections) != 0 {
				t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
			}
			if len(decision.Events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(decision.Events))
			}
			evt := decision.Events[0]
			if evt.Type != event.Type(tt.wantType) {
				t.Fatalf("event type = %s, want %s", evt.Type, tt.wantType)
			}
			if evt.EntityType != tt.wantEntityType {
				t.Fatalf("event entity type = %s, want %s", evt.EntityType, tt.wantEntityType)
			}
			if evt.EntityID != tt.wantEntityID {
				t.Fatalf("event entity id = %s, want %s", evt.EntityID, tt.wantEntityID)
			}
			if !evt.Timestamp.Equal(now) {
				t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
			}
			if evt.ActorType != event.ActorTypeSystem {
				t.Fatalf("event actor type = %s, want %s", evt.ActorType, event.ActorTypeSystem)
			}
			if evt.SessionID != "sess-1" {
				t.Fatalf("event session id = %s, want %s", evt.SessionID, "sess-1")
			}
		})
	}
}

func TestDecideActionCommands_PreservesSystemMetadata(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	decision := Decide(State{}, command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.roll.resolve"),
		ActorType:     command.ActorTypeSystem,
		SessionID:     "sess-1",
		SystemID:      "daggerheart",
		SystemVersion: "v1",
		PayloadJSON:   []byte(`{"request_id":"req-1","roll_seq":1}`),
	}, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	evt := decision.Events[0]
	if evt.SystemID != "daggerheart" {
		t.Fatalf("event system id = %s, want %s", evt.SystemID, "daggerheart")
	}
	if evt.SystemVersion != "v1" {
		t.Fatalf("event system version = %s, want %s", evt.SystemVersion, "v1")
	}
}

func TestDecideActionCommands_RejectsUnsupportedType(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("action.not_registered"),
		ActorType:  command.ActorTypeSystem,
	}, time.Now)

	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "COMMAND_TYPE_UNSUPPORTED" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "COMMAND_TYPE_UNSUPPORTED")
	}
}

func TestDecideActionCommands_RejectsDuplicateOutcomeApply(t *testing.T) {
	decision := Decide(State{
		AppliedOutcomes: map[uint64]struct{}{
			2: {},
		},
	}, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("action.outcome.apply"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"request_id":"req-2","roll_seq":2}`),
	}, time.Now)

	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "OUTCOME_ALREADY_APPLIED" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "OUTCOME_ALREADY_APPLIED")
	}
}
