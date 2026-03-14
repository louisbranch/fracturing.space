package invite

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestDecideInviteDeclineEmitsInviteDeclinedEvent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	decision := Decide(State{Created: true, Status: statusPending}, command.Command{
		CampaignID:  "camp-1",
		Type:        CommandTypeDecline,
		ActorType:   command.ActorTypeParticipant,
		ActorID:     "p-1",
		PayloadJSON: []byte(`{"invite_id":"inv-1","user_id":" user-1 "}`),
	}, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != EventTypeDeclined {
		t.Fatalf("event type = %s, want %s", evt.Type, EventTypeDeclined)
	}
	if evt.EntityType != "invite" || evt.EntityID != "inv-1" {
		t.Fatalf("event routing = (%s,%s), want (invite,inv-1)", evt.EntityType, evt.EntityID)
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload DeclinePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.InviteID != "inv-1" || payload.UserID != "user-1" {
		t.Fatalf("payload = %+v, want trimmed decline payload", payload)
	}
}

func TestDecideInviteDeclineRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   State
		payload string
		code    string
	}{
		{name: "not created", state: State{}, payload: `{"invite_id":"inv-1","user_id":"user-1"}`, code: rejectionCodeInviteNotCreated},
		{name: "invalid status", state: State{Created: true, Status: statusClaimed}, payload: `{"invite_id":"inv-1","user_id":"user-1"}`, code: rejectionCodeInviteStatusInvalid},
		{name: "missing invite id", state: State{Created: true, Status: statusPending}, payload: `{"user_id":"user-1"}`, code: rejectionCodeInviteIDRequired},
		{name: "missing user id", state: State{Created: true, Status: statusPending}, payload: `{"invite_id":"inv-1"}`, code: rejectionCodeInviteUserIDRequired},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decision := Decide(tc.state, command.Command{
				CampaignID:  "camp-1",
				Type:        CommandTypeDecline,
				PayloadJSON: []byte(tc.payload),
			}, nil)
			if len(decision.Events) != 0 {
				t.Fatalf("expected no events, got %d", len(decision.Events))
			}
			if len(decision.Rejections) != 1 || decision.Rejections[0].Code != tc.code {
				t.Fatalf("rejections = %+v, want code %s", decision.Rejections, tc.code)
			}
		})
	}
}

func TestFoldInviteDeclinedUpdatesStatus(t *testing.T) {
	t.Parallel()

	updated, err := Fold(State{Created: true, InviteID: "inv-1", Status: statusPending}, event.Event{
		Type:        EventTypeDeclined,
		PayloadJSON: []byte(`{"invite_id":"inv-1","user_id":"user-1"}`),
	})
	if err != nil {
		t.Fatalf("Fold() error = %v", err)
	}
	if updated.Status != statusDeclined {
		t.Fatalf("status = %s, want %s", updated.Status, statusDeclined)
	}
}

func TestRegisterCommandsAndEventsValidateDeclinePayload(t *testing.T) {
	t.Parallel()

	commandRegistry := command.NewRegistry()
	if err := RegisterCommands(commandRegistry); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if _, err := commandRegistry.ValidateForDecision(command.Command{
		CampaignID:  "camp-1",
		Type:        CommandTypeDecline,
		PayloadJSON: []byte(`{"invite_id":"inv-1","user_id":"user-1"}`),
	}); err != nil {
		t.Fatalf("decline command rejected: %v", err)
	}

	eventRegistry := event.NewRegistry()
	if err := RegisterEvents(eventRegistry); err != nil {
		t.Fatalf("register events: %v", err)
	}
	if _, err := eventRegistry.ValidateForAppend(event.Event{
		CampaignID:  "camp-1",
		Type:        EventTypeDeclined,
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   event.ActorTypeParticipant,
		ActorID:     "p-1",
		EntityType:  "invite",
		EntityID:    "inv-1",
		PayloadJSON: []byte(`{"invite_id":"inv-1","user_id":"user-1"}`),
	}); err != nil {
		t.Fatalf("decline event rejected: %v", err)
	}
}
