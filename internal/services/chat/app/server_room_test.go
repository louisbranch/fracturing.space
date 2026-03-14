package server

import (
	"testing"
	"time"
)

func TestCampaignRoomAIRelayReadyWithValidGrant(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("grant-token", 1, time.Now().UTC().Add(time.Minute))

	if !room.aiRelayReady() {
		t.Fatal("expected ai relay to be ready with a non-expired grant")
	}
}

func TestCampaignRoomAIRelayReadyClearsExpiredGrant(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("grant-token", 1, time.Now().UTC().Add(time.Second))

	if room.aiRelayReady() {
		t.Fatal("expected ai relay to be not ready for an expired/near-expiry grant")
	}
	if got := room.aiSessionGrantValue(); got != "" {
		t.Fatalf("grant token = %q, want empty", got)
	}
}

func TestCampaignRoomPendingAITurnSubmissionTracksOnlyNewParticipantMessages(t *testing.T) {
	room := newCampaignRoom("camp-1")
	room.setSessionID("sess-1")

	first, _, _ := room.appendMessage(messageActor{
		ParticipantID: "part-1",
		Name:          "Ari",
		Mode:          "participant",
	}, "first plan", "cli-1", chatDefaultStreamID("camp-1"))
	second, _, _ := room.appendMessage(messageActor{
		ParticipantID: "part-1",
		CharacterID:   "char-1",
		Name:          "Vera",
		Mode:          "character",
	}, "check the door", "cli-2", "scene:scene-1:character")
	_, _, _ = room.appendMessage(messageActor{
		ParticipantID: "part-1",
		Name:          "Ari",
		Mode:          "participant",
	}, "control chatter", "cli-3", chatControlStreamID("camp-1"))

	submission, ok := room.pendingAITurnSubmission("ready for a ruling")
	if !ok {
		t.Fatal("expected pending submission")
	}
	if submission.correlationMessageID != second.MessageID {
		t.Fatalf("correlation message id = %q, want %q", submission.correlationMessageID, second.MessageID)
	}
	if submission.highestSequenceID != second.SequenceID {
		t.Fatalf("highest sequence id = %d, want %d", submission.highestSequenceID, second.SequenceID)
	}
	if submission.body == "" || submission.body == "ready for a ruling" {
		t.Fatalf("submission body = %q, expected transcript and reason", submission.body)
	}

	room.markAITurnSubmitted(submission.highestSequenceID)
	nextSubmission, ok := room.pendingAITurnSubmission("")
	if ok {
		t.Fatalf("expected no pending submission after mark, got %+v", nextSubmission)
	}

	third, _, _ := room.appendMessage(messageActor{
		ParticipantID: "part-1",
		Name:          "Ari",
		Mode:          "participant",
	}, "open the door", "cli-4", chatDefaultStreamID("camp-1"))
	submission, ok = room.pendingAITurnSubmission("")
	if !ok {
		t.Fatal("expected new pending submission")
	}
	if submission.correlationMessageID != third.MessageID {
		t.Fatalf("correlation message id = %q, want %q", submission.correlationMessageID, third.MessageID)
	}
	if submission.highestSequenceID != third.SequenceID {
		t.Fatalf("highest sequence id = %d, want %d", submission.highestSequenceID, third.SequenceID)
	}
	if submission.body == "" || submission.body == first.Body {
		t.Fatalf("submission body = %q, expected only new transcript", submission.body)
	}
}
