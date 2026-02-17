package participant

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestFoldParticipantJoinedSetsFields(t *testing.T) {
	state := State{}
	updated := Fold(state, event.Event{
		Type:        event.Type("participant.joined"),
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"u-1","name":"Alice","role":"player","controller":"human","campaign_access":"member"}`),
	})
	if !updated.Joined {
		t.Fatal("expected participant to be joined")
	}
	if updated.ParticipantID != "p-1" {
		t.Fatalf("participant id = %s, want %s", updated.ParticipantID, "p-1")
	}
	if updated.UserID != "u-1" {
		t.Fatalf("user id = %s, want %s", updated.UserID, "u-1")
	}
	if updated.Name != "Alice" {
		t.Fatalf("display name = %s, want %s", updated.Name, "Alice")
	}
	if updated.Role != "player" {
		t.Fatalf("role = %s, want %s", updated.Role, "player")
	}
	if updated.Controller != "human" {
		t.Fatalf("controller = %s, want %s", updated.Controller, "human")
	}
	if updated.CampaignAccess != "member" {
		t.Fatalf("campaign access = %s, want %s", updated.CampaignAccess, "member")
	}
}

func TestFoldParticipantUpdatedSetsFields(t *testing.T) {
	state := State{Joined: true, ParticipantID: "p-1", Name: "Old", Role: "gm"}
	updated := Fold(state, event.Event{
		Type:        event.Type("participant.updated"),
		PayloadJSON: []byte(`{"participant_id":"p-1","fields":{"name":"Alice","role":"player","controller":"human","campaign_access":"member"}}`),
	})
	if updated.Name != "Alice" {
		t.Fatalf("display name = %s, want %s", updated.Name, "Alice")
	}
	if updated.Role != "player" {
		t.Fatalf("role = %s, want %s", updated.Role, "player")
	}
	if updated.Controller != "human" {
		t.Fatalf("controller = %s, want %s", updated.Controller, "human")
	}
	if updated.CampaignAccess != "member" {
		t.Fatalf("campaign access = %s, want %s", updated.CampaignAccess, "member")
	}
}

func TestFoldParticipantLeftMarksLeft(t *testing.T) {
	state := State{Joined: true, ParticipantID: "p-1"}
	updated := Fold(state, event.Event{
		Type:        event.Type("participant.left"),
		PayloadJSON: []byte(`{"participant_id":"p-1","reason":"done"}`),
	})
	if updated.Joined {
		t.Fatal("expected participant to be marked not joined")
	}
	if !updated.Left {
		t.Fatal("expected participant to be marked left")
	}
}

func TestFoldParticipantBoundSetsUserID(t *testing.T) {
	state := State{Joined: true, ParticipantID: "p-1"}
	updated := Fold(state, event.Event{
		Type:        event.Type("participant.bound"),
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"u-2"}`),
	})
	if updated.UserID != "u-2" {
		t.Fatalf("user id = %s, want %s", updated.UserID, "u-2")
	}
}

func TestFoldParticipantUnboundClearsUserID(t *testing.T) {
	state := State{Joined: true, ParticipantID: "p-1", UserID: "u-1"}
	updated := Fold(state, event.Event{
		Type:        event.Type("participant.unbound"),
		PayloadJSON: []byte(`{"participant_id":"p-1","user_id":"u-1"}`),
	})
	if updated.UserID != "" {
		t.Fatalf("user id = %s, want %s", updated.UserID, "")
	}
}

func TestFoldSeatReassignedUpdatesUserID(t *testing.T) {
	state := State{Joined: true, ParticipantID: "p-1", UserID: "u-1"}
	updated := Fold(state, event.Event{
		Type:        event.Type("seat.reassigned"),
		PayloadJSON: []byte(`{"participant_id":"p-1","prior_user_id":"u-1","user_id":"u-2"}`),
	})
	if updated.UserID != "u-2" {
		t.Fatalf("user id = %s, want %s", updated.UserID, "u-2")
	}
}
