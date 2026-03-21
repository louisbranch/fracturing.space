package protocol

import (
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

func TestTranscriptMessageNormalizesTranscriptFields(t *testing.T) {
	t.Parallel()

	got := TranscriptMessage(transcript.Message{
		MessageID:  " msg-1 ",
		CampaignID: " c1 ",
		SessionID:  " s1 ",
		SequenceID: 7,
		SentAt:     " 2026-03-13T12:00:00Z ",
		Actor: transcript.MessageActor{
			ParticipantID: " p1 ",
			Name:          " Avery ",
		},
		Body:            " hello ",
		ClientMessageID: " cm-1 ",
	})

	if got.MessageID != "msg-1" {
		t.Fatalf("MessageID = %q, want %q", got.MessageID, "msg-1")
	}
	if got.CampaignID != "c1" || got.SessionID != "s1" {
		t.Fatalf("scope = (%q, %q), want (%q, %q)", got.CampaignID, got.SessionID, "c1", "s1")
	}
	if got.Actor.ParticipantID != "p1" || got.Actor.Name != "Avery" {
		t.Fatalf("actor = %#v", got.Actor)
	}
	if got.Body != "hello" || got.ClientMessageID != "cm-1" {
		t.Fatalf("message = %#v", got)
	}
}

func TestTranscriptMessagesPreservesOrdering(t *testing.T) {
	t.Parallel()

	got := TranscriptMessages([]transcript.Message{
		{MessageID: "m1", SequenceID: 1},
		{MessageID: "m2", SequenceID: 2},
	})

	if len(got) != 2 {
		t.Fatalf("len = %d, want %d", len(got), 2)
	}
	if got[0].MessageID != "m1" || got[1].MessageID != "m2" {
		t.Fatalf("messages = %#v", got)
	}
}

func TestInteractionStateFromGameStateBuildsPlayOwnedDTO(t *testing.T) {
	t.Parallel()

	got := InteractionStateFromGameState(&gamev1.InteractionState{
		CampaignId:   " c1 ",
		CampaignName: " Guildhouse ",
		Viewer: &gamev1.InteractionViewer{
			ParticipantId: " p1 ",
			Name:          " Avery ",
			Role:          gamev1.ParticipantRole_PLAYER,
		},
		ActiveSession: &gamev1.InteractionSession{
			SessionId: " s1 ",
			Name:      " Session One ",
		},
	})

	if got.CampaignID != "c1" || got.CampaignName != "Guildhouse" {
		t.Fatalf("state = %#v", got)
	}
	if got.Viewer == nil || got.Viewer.ParticipantID != "p1" || got.Viewer.Name != "Avery" || got.Viewer.Role != "player" {
		t.Fatalf("viewer = %#v", got.Viewer)
	}
	if got.ActiveSession == nil || got.ActiveSession.SessionID != "s1" || got.ActiveSession.Name != "Session One" {
		t.Fatalf("active session = %#v", got.ActiveSession)
	}
}
