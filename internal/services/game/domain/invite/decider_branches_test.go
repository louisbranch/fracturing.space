package invite

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

func TestDecideInviteCreate_Rejections(t *testing.T) {
	tests := []struct {
		name    string
		state   State
		payload string
		code    string
	}{
		{name: "already created", state: State{Created: true}, payload: `{"invite_id":"inv-1","participant_id":"p-1"}`, code: rejectionCodeInviteAlreadyExists},
		{name: "missing invite id", payload: `{"participant_id":"p-1"}`, code: rejectionCodeInviteIDRequired},
		{name: "missing participant id", payload: `{"invite_id":"inv-1"}`, code: rejectionCodeInviteParticipantNeeded},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := Decide(tc.state, command.Command{
				CampaignID:  "camp-1",
				Type:        CommandTypeCreate,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(tc.payload),
			}, time.Now)
			if len(decision.Events) != 0 {
				t.Fatalf("expected no events, got %d", len(decision.Events))
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
			}
			if decision.Rejections[0].Code != tc.code {
				t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, tc.code)
			}
		})
	}
}

func TestDecideInviteClaim_Rejections(t *testing.T) {
	tests := []struct {
		name    string
		state   State
		payload string
		code    string
	}{
		{name: "invite not created", state: State{}, payload: `{"invite_id":"inv-1","participant_id":"p-1","user_id":"u-1","jti":"j-1"}`, code: rejectionCodeInviteNotCreated},
		{name: "invalid status", state: State{Created: true, Status: statusRevoked}, payload: `{"invite_id":"inv-1","participant_id":"p-1","user_id":"u-1","jti":"j-1"}`, code: rejectionCodeInviteStatusInvalid},
		{name: "missing invite id", state: State{Created: true, Status: statusPending}, payload: `{"participant_id":"p-1","user_id":"u-1","jti":"j-1"}`, code: rejectionCodeInviteIDRequired},
		{name: "missing participant id", state: State{Created: true, Status: statusPending}, payload: `{"invite_id":"inv-1","user_id":"u-1","jti":"j-1"}`, code: rejectionCodeInviteParticipantNeeded},
		{name: "missing user id", state: State{Created: true, Status: statusPending}, payload: `{"invite_id":"inv-1","participant_id":"p-1","jti":"j-1"}`, code: rejectionCodeInviteUserIDRequired},
		{name: "missing jwt id", state: State{Created: true, Status: statusPending}, payload: `{"invite_id":"inv-1","participant_id":"p-1","user_id":"u-1"}`, code: rejectionCodeInviteJWTRequired},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := Decide(tc.state, command.Command{
				CampaignID:  "camp-1",
				Type:        CommandTypeClaim,
				ActorType:   command.ActorTypeParticipant,
				ActorID:     "p-1",
				PayloadJSON: []byte(tc.payload),
			}, time.Now)
			if len(decision.Events) != 0 {
				t.Fatalf("expected no events, got %d", len(decision.Events))
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
			}
			if decision.Rejections[0].Code != tc.code {
				t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, tc.code)
			}
		})
	}
}

func TestDecideInviteRevokeAndUpdate_Rejections(t *testing.T) {
	tests := []struct {
		name    string
		state   State
		cmdType command.Type
		payload string
		code    string
	}{
		{name: "revoke not created", cmdType: CommandTypeRevoke, payload: `{"invite_id":"inv-1"}`, code: rejectionCodeInviteNotCreated},
		{name: "revoke invalid status", state: State{Created: true, Status: statusClaimed}, cmdType: CommandTypeRevoke, payload: `{"invite_id":"inv-1"}`, code: rejectionCodeInviteStatusInvalid},
		{name: "revoke missing invite id", state: State{Created: true, Status: statusPending}, cmdType: CommandTypeRevoke, payload: `{}`, code: rejectionCodeInviteIDRequired},
		{name: "update not created", cmdType: CommandTypeUpdate, payload: `{"invite_id":"inv-1","status":"pending"}`, code: rejectionCodeInviteNotCreated},
		{name: "update missing invite id", state: State{Created: true, Status: statusPending}, cmdType: CommandTypeUpdate, payload: `{"status":"pending"}`, code: rejectionCodeInviteIDRequired},
		{name: "update invalid status", state: State{Created: true, Status: statusPending}, cmdType: CommandTypeUpdate, payload: `{"invite_id":"inv-1","status":"invalid"}`, code: rejectionCodeInviteStatusInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := Decide(tc.state, command.Command{
				CampaignID:  "camp-1",
				Type:        tc.cmdType,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(tc.payload),
			}, time.Now)
			if len(decision.Events) != 0 {
				t.Fatalf("expected no events, got %d", len(decision.Events))
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
			}
			if decision.Rejections[0].Code != tc.code {
				t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, tc.code)
			}
		})
	}
}

func TestDecideInviteCreateAndClaim_IgnoreCommandEntityRouting(t *testing.T) {
	create := Decide(State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        CommandTypeCreate,
		ActorType:   command.ActorTypeSystem,
		EntityType:  "override",
		EntityID:    "override",
		PayloadJSON: []byte(`{"invite_id":"inv-entity","participant_id":"p-1"}`),
	}, time.Now)
	if len(create.Events) != 1 {
		t.Fatalf("expected 1 create event, got %d", len(create.Events))
	}
	if create.Events[0].EntityType != "invite" || create.Events[0].EntityID != "inv-entity" {
		t.Fatalf("create routing = (%s,%s), want (invite,inv-entity)", create.Events[0].EntityType, create.Events[0].EntityID)
	}

	claim := Decide(State{Created: true, Status: statusPending}, command.Command{
		CampaignID:  "camp-1",
		Type:        CommandTypeClaim,
		ActorType:   command.ActorTypeParticipant,
		EntityType:  "override",
		EntityID:    "override",
		PayloadJSON: []byte(`{"invite_id":"inv-entity","participant_id":"p-1","user_id":"u-1","jti":"j-1"}`),
	}, time.Now)
	if len(claim.Events) != 1 {
		t.Fatalf("expected 1 claim event, got %d", len(claim.Events))
	}
	if claim.Events[0].EntityType != "invite" || claim.Events[0].EntityID != "inv-entity" {
		t.Fatalf("claim routing = (%s,%s), want (invite,inv-entity)", claim.Events[0].EntityType, claim.Events[0].EntityID)
	}
}
