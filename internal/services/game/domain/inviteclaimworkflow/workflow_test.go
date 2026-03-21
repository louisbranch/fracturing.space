package inviteclaimworkflow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestDecide_ClaimsInviteAfterParticipantBind(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	payloadJSON, err := json.Marshal(ClaimBindPayload{
		InviteID:      "invite-1",
		ParticipantID: "participant-1",
		UserID:        "user-1",
		JWTID:         "jwt-1",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	state := aggregate.NewState()
	state.Participants[ids.ParticipantID("participant-1")] = participant.State{
		Joined:        true,
		ParticipantID: "participant-1",
	}
	state.Invites[ids.InviteID("invite-1")] = invite.State{
		Created:       true,
		InviteID:      "invite-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}

	decision := Decide(state, command.Command{
		CampaignID:   "campaign-1",
		Type:         CommandTypeClaimBind,
		ActorType:    command.ActorTypeParticipant,
		ActorID:      "participant-actor",
		RequestID:    "req-1",
		InvocationID: "inv-1",
		EntityType:   "invite",
		EntityID:     "invite-1",
		PayloadJSON:  payloadJSON,
	}, func() time.Time { return now })

	if len(decision.Rejections) > 0 {
		t.Fatalf("rejections = %v, want none", decision.Rejections)
	}
	if len(decision.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(decision.Events))
	}
	if decision.Events[0].Type != participant.EventTypeBound {
		t.Fatalf("first event type = %s, want %s", decision.Events[0].Type, participant.EventTypeBound)
	}
	if decision.Events[1].Type != invite.EventTypeClaimed {
		t.Fatalf("second event type = %s, want %s", decision.Events[1].Type, invite.EventTypeClaimed)
	}
	if !decision.Events[0].Timestamp.Equal(now) || !decision.Events[1].Timestamp.Equal(now) {
		t.Fatalf("expected shared decision timestamp, got %v and %v", decision.Events[0].Timestamp, decision.Events[1].Timestamp)
	}
	if decision.Events[0].ActorType != event.ActorTypeParticipant || decision.Events[1].ActorType != event.ActorTypeParticipant {
		t.Fatalf("actor types = %s/%s, want participant/participant", decision.Events[0].ActorType, decision.Events[1].ActorType)
	}
}

func TestDecide_StopsOnParticipantRejection(t *testing.T) {
	payloadJSON, err := json.Marshal(ClaimBindPayload{
		InviteID:      "invite-1",
		ParticipantID: "participant-1",
		UserID:        "user-1",
		JWTID:         "jwt-1",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	state := aggregate.NewState()
	state.Participants[ids.ParticipantID("participant-1")] = participant.State{
		Joined:        true,
		ParticipantID: "participant-1",
		UserID:        "user-existing",
	}
	state.Invites[ids.InviteID("invite-1")] = invite.State{
		Created:       true,
		InviteID:      "invite-1",
		ParticipantID: "participant-1",
		Status:        invite.StatusPending,
	}

	decision := Decide(state, command.Command{
		CampaignID:  "campaign-1",
		Type:        CommandTypeClaimBind,
		ActorType:   command.ActorTypeSystem,
		EntityType:  "invite",
		EntityID:    "invite-1",
		PayloadJSON: payloadJSON,
	}, time.Now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if len(decision.Events) != 0 {
		t.Fatalf("events = %d, want 0", len(decision.Events))
	}
}
