package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

func TestStoreAppendMessageMaintainsSequenceAndIdempotency(t *testing.T) {
	t.Parallel()

	store, err := Open(filepath.Join(t.TempDir(), "play.sqlite"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	baseTime := time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC)
	callCount := 0
	store.now = func() time.Time {
		callCount++
		return baseTime.Add(time.Duration(callCount) * time.Second)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	first, duplicate, err := store.AppendMessage(ctx, "camp-1", "sess-1", transcript.MessageActor{ParticipantID: "p1", Name: "Avery"}, "hello", "cli-1")
	if err != nil {
		t.Fatalf("AppendMessage(first) error = %v", err)
	}
	if duplicate {
		t.Fatal("first append unexpectedly marked duplicate")
	}
	if first.SequenceID != 1 {
		t.Fatalf("first.SequenceID = %d, want 1", first.SequenceID)
	}

	again, duplicate, err := store.AppendMessage(ctx, "camp-1", "sess-1", transcript.MessageActor{ParticipantID: "p1", Name: "Avery"}, "hello", "cli-1")
	if err != nil {
		t.Fatalf("AppendMessage(duplicate) error = %v", err)
	}
	if !duplicate {
		t.Fatal("duplicate append was not detected")
	}
	if again.MessageID != first.MessageID || again.SequenceID != first.SequenceID {
		t.Fatalf("duplicate append returned %#v, want %#v", again, first)
	}

	second, duplicate, err := store.AppendMessage(ctx, "camp-1", "sess-1", transcript.MessageActor{ParticipantID: "p2", Name: "Bo"}, "second", "cli-2")
	if err != nil {
		t.Fatalf("AppendMessage(second) error = %v", err)
	}
	if duplicate {
		t.Fatal("second append unexpectedly marked duplicate")
	}
	if second.SequenceID != 2 {
		t.Fatalf("second.SequenceID = %d, want 2", second.SequenceID)
	}

	latest, err := store.LatestSequence(ctx, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("LatestSequence() error = %v", err)
	}
	if latest != 2 {
		t.Fatalf("LatestSequence() = %d, want 2", latest)
	}

	history, err := store.HistoryAfter(ctx, "camp-1", "sess-1", 0)
	if err != nil {
		t.Fatalf("HistoryAfter() error = %v", err)
	}
	if len(history) != 2 || history[0].SequenceID != 1 || history[1].SequenceID != 2 {
		t.Fatalf("HistoryAfter() = %#v, want sequence ids [1 2]", history)
	}

	before, err := store.HistoryBefore(ctx, "camp-1", "sess-1", 3, 10)
	if err != nil {
		t.Fatalf("HistoryBefore() error = %v", err)
	}
	if len(before) != 2 || before[0].SequenceID != 1 || before[1].SequenceID != 2 {
		t.Fatalf("HistoryBefore() = %#v, want sequence ids [1 2]", before)
	}
}
