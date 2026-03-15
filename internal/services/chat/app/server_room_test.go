package server

import (
	"encoding/json"
	"io"
	"testing"
)

func TestInMemoryTranscriptStoreAppendsPerSessionRoom(t *testing.T) {
	t.Parallel()

	store := newInMemoryTranscriptStore()

	msgA, duplicate := store.AppendMessage("camp-1", "sess-a", messageActor{ParticipantID: "part-1", Name: "Ari"}, "hello", "cli-1")
	if duplicate {
		t.Fatal("first append unexpectedly marked duplicate")
	}
	msgB, duplicate := store.AppendMessage("camp-1", "sess-b", messageActor{ParticipantID: "part-2", Name: "Bo"}, "hi", "cli-2")
	if duplicate {
		t.Fatal("second append unexpectedly marked duplicate")
	}

	if msgA.SessionID != "sess-a" || msgA.SequenceID != 1 {
		t.Fatalf("unexpected first message: %+v", msgA)
	}
	if msgB.SessionID != "sess-b" || msgB.SequenceID != 1 {
		t.Fatalf("unexpected second message: %+v", msgB)
	}
	if got := store.LatestSequence("camp-1", "sess-a"); got != 1 {
		t.Fatalf("LatestSequence(sess-a) = %d, want 1", got)
	}
	if got := store.LatestSequence("camp-1", "sess-b"); got != 1 {
		t.Fatalf("LatestSequence(sess-b) = %d, want 1", got)
	}
}

func TestInMemoryTranscriptStoreDeduplicatesClientMessageID(t *testing.T) {
	t.Parallel()

	store := newInMemoryTranscriptStore()

	first, duplicate := store.AppendMessage("camp-1", "sess-1", messageActor{ParticipantID: "part-1", Name: "Ari"}, "hello", "cli-1")
	if duplicate {
		t.Fatal("first append unexpectedly marked duplicate")
	}
	second, duplicate := store.AppendMessage("camp-1", "sess-1", messageActor{ParticipantID: "part-1", Name: "Ari"}, "changed", "cli-1")
	if !duplicate {
		t.Fatal("second append should be duplicate")
	}
	if second.MessageID != first.MessageID || second.Body != first.Body {
		t.Fatalf("duplicate append returned %+v, want original %+v", second, first)
	}

	history := store.HistoryBefore("camp-1", "sess-1", 2, 10)
	if len(history) != 1 {
		t.Fatalf("HistoryBefore() len = %d, want 1", len(history))
	}
	if history[0].Body != "hello" {
		t.Fatalf("HistoryBefore()[0].Body = %q, want hello", history[0].Body)
	}
}

func TestSessionRoomJoinWithHistoryReturnsBacklogBeforeLiveSubscription(t *testing.T) {
	t.Parallel()

	store := newInMemoryTranscriptStore()
	room := newSessionRoom("camp-1", "sess-1", store)

	first, duplicate, _ := room.appendMessage(messageActor{ParticipantID: "part-1", Name: "Ari"}, "hello", "cli-1")
	if duplicate {
		t.Fatal("first append unexpectedly marked duplicate")
	}

	session := newWSSession("user-1", newWSPeer(json.NewEncoder(io.Discard)))
	latest, history := room.joinWithHistory(session, 0)
	if latest != first.SequenceID {
		t.Fatalf("latest = %d, want %d", latest, first.SequenceID)
	}
	if len(history) != 1 || history[0].SequenceID != first.SequenceID {
		t.Fatalf("history = %#v", history)
	}

	second, duplicate, subscribers := room.appendMessage(messageActor{ParticipantID: "part-2", Name: "Bo"}, "hi", "cli-2")
	if duplicate {
		t.Fatal("second append unexpectedly marked duplicate")
	}
	if len(subscribers) != 1 || subscribers[0] != session.peer {
		t.Fatalf("subscribers = %#v", subscribers)
	}
	if latest != first.SequenceID || second.SequenceID != first.SequenceID+1 {
		t.Fatalf("sequence progression = (%d,%d)", latest, second.SequenceID)
	}
}
