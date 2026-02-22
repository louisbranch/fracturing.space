package invite

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestDecideInviteCreate_EmitsInviteCreatedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","recipient_user_id":" user-1 ","created_by_participant_id":" gm-1 "}`),
	}

	decision := Decide(State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.CampaignID != "camp-1" {
		t.Fatalf("event campaign id = %s, want %s", evt.CampaignID, "camp-1")
	}
	if evt.Type != event.Type("invite.created") {
		t.Fatalf("event type = %s, want %s", evt.Type, "invite.created")
	}
	if evt.EntityType != "invite" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "invite")
	}
	if evt.EntityID != "inv-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "inv-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}
	if evt.ActorType != event.ActorTypeSystem {
		t.Fatalf("event actor type = %s, want %s", evt.ActorType, event.ActorTypeSystem)
	}

	var payload CreatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.InviteID != "inv-1" {
		t.Fatalf("payload invite id = %s, want %s", payload.InviteID, "inv-1")
	}
	if payload.ParticipantID != "p-1" {
		t.Fatalf("payload participant id = %s, want %s", payload.ParticipantID, "p-1")
	}
	if payload.RecipientUserID != "user-1" {
		t.Fatalf("payload recipient user id = %s, want %s", payload.RecipientUserID, "user-1")
	}
	if payload.CreatedByParticipantID != "gm-1" {
		t.Fatalf("payload created by participant id = %s, want %s", payload.CreatedByParticipantID, "gm-1")
	}
	if payload.Status != "pending" {
		t.Fatalf("payload status = %s, want %s", payload.Status, "pending")
	}
}

func TestDecideInviteClaim_EmitsInviteClaimedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.claim"),
		ActorType:   command.ActorTypeParticipant,
		ActorID:     "p-1",
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","user_id":" user-1 ","jti":" jwt-1 "}`),
	}

	decision := Decide(State{Created: true, Status: "pending"}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("invite.claimed") {
		t.Fatalf("event type = %s, want %s", evt.Type, "invite.claimed")
	}
	if evt.EntityType != "invite" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "invite")
	}
	if evt.EntityID != "inv-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "inv-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}
	if evt.ActorType != event.ActorTypeParticipant {
		t.Fatalf("event actor type = %s, want %s", evt.ActorType, event.ActorTypeParticipant)
	}
	if evt.ActorID != "p-1" {
		t.Fatalf("event actor id = %s, want %s", evt.ActorID, "p-1")
	}

	var payload ClaimPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.InviteID != "inv-1" {
		t.Fatalf("payload invite id = %s, want %s", payload.InviteID, "inv-1")
	}
	if payload.ParticipantID != "p-1" {
		t.Fatalf("payload participant id = %s, want %s", payload.ParticipantID, "p-1")
	}
	if payload.UserID != "user-1" {
		t.Fatalf("payload user id = %s, want %s", payload.UserID, "user-1")
	}
	if payload.JWTID != "jwt-1" {
		t.Fatalf("payload jti = %s, want %s", payload.JWTID, "jwt-1")
	}
}

func TestDecideInviteRevoke_EmitsInviteRevokedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.revoke"),
		ActorType:   command.ActorTypeParticipant,
		ActorID:     "p-1",
		PayloadJSON: []byte(`{"invite_id":"inv-1"}`),
	}

	decision := Decide(State{Created: true, Status: "pending"}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("invite.revoked") {
		t.Fatalf("event type = %s, want %s", evt.Type, "invite.revoked")
	}
	if evt.EntityType != "invite" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "invite")
	}
	if evt.EntityID != "inv-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "inv-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}
	if evt.ActorType != event.ActorTypeParticipant {
		t.Fatalf("event actor type = %s, want %s", evt.ActorType, event.ActorTypeParticipant)
	}
	if evt.ActorID != "p-1" {
		t.Fatalf("event actor id = %s, want %s", evt.ActorID, "p-1")
	}

	var payload RevokePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.InviteID != "inv-1" {
		t.Fatalf("payload invite id = %s, want %s", payload.InviteID, "inv-1")
	}
}

func TestDecideInviteUpdate_EmitsInviteUpdatedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"invite_id":"inv-1","status":"REVOKED"}`),
	}

	decision := Decide(State{Created: true, Status: "pending"}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("invite.updated") {
		t.Fatalf("event type = %s, want %s", evt.Type, "invite.updated")
	}
	if evt.EntityType != "invite" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "invite")
	}
	if evt.EntityID != "inv-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "inv-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload UpdatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.InviteID != "inv-1" {
		t.Fatalf("payload invite id = %s, want %s", payload.InviteID, "inv-1")
	}
	if payload.Status != "revoked" {
		t.Fatalf("payload status = %s, want %s", payload.Status, "revoked")
	}
}

func TestDecide_UnrecognizedCommandTypeRejected(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("invite.nonexistent"),
	}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "COMMAND_TYPE_UNSUPPORTED" {
		t.Fatalf("rejection code = %s, want COMMAND_TYPE_UNSUPPORTED", decision.Rejections[0].Code)
	}
}

func TestDecide_MalformedCreatePayloadRejected(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.create"),
		PayloadJSON: []byte(`{corrupt`),
	}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "PAYLOAD_DECODE_FAILED" {
		t.Fatalf("rejection code = %s, want PAYLOAD_DECODE_FAILED", decision.Rejections[0].Code)
	}
}
