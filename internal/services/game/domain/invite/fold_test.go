package invite

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestFoldInviteCreatedSetsFields(t *testing.T) {
	state := State{}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("invite.created"),
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","recipient_user_id":"user-1","created_by_participant_id":"gm-1","status":"pending"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.Created {
		t.Fatal("expected invite to be marked created")
	}
	if updated.InviteID != "inv-1" {
		t.Fatalf("invite id = %s, want %s", updated.InviteID, "inv-1")
	}
	if updated.ParticipantID != "p-1" {
		t.Fatalf("participant id = %s, want %s", updated.ParticipantID, "p-1")
	}
	if updated.RecipientUserID != "user-1" {
		t.Fatalf("recipient user id = %s, want %s", updated.RecipientUserID, "user-1")
	}
	if updated.CreatedByParticipantID != "gm-1" {
		t.Fatalf("created by participant id = %s, want %s", updated.CreatedByParticipantID, "gm-1")
	}
	if updated.Status != "pending" {
		t.Fatalf("status = %s, want %s", updated.Status, "pending")
	}
}

func TestFoldInviteClaimedUpdatesStatus(t *testing.T) {
	state := State{Created: true, InviteID: "inv-1", Status: "pending"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("invite.claimed"),
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","user_id":"user-1","jti":"jwt-1"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != "claimed" {
		t.Fatalf("status = %s, want %s", updated.Status, "claimed")
	}
}

func TestFoldInviteRevokedUpdatesStatus(t *testing.T) {
	state := State{Created: true, InviteID: "inv-1", Status: "pending"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("invite.revoked"),
		PayloadJSON: []byte(`{"invite_id":"inv-1"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != "revoked" {
		t.Fatalf("status = %s, want %s", updated.Status, "revoked")
	}
}

func TestFoldInviteUpdatedSetsStatus(t *testing.T) {
	state := State{Created: true, InviteID: "inv-1", Status: "pending"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("invite.updated"),
		PayloadJSON: []byte(`{"invite_id":"inv-1","status":"revoked"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != "revoked" {
		t.Fatalf("status = %s, want %s", updated.Status, "revoked")
	}
}

func TestFoldInviteCreatedNormalizesStatus(t *testing.T) {
	state := State{}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("invite.created"),
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","status":"PENDING"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != "pending" {
		t.Fatalf("status = %s, want %s", updated.Status, "pending")
	}
}
