// Package transcripttest provides reusable transcript store contract tests for
// concrete adapters.
package transcripttest

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

// Store is the subset of the transcript seam exercised by adapter contract
// tests. Concrete stores may optionally also implement io.Closer semantics.
type Store = transcript.Store

// Factory creates one fresh transcript store for a contract subtest.
type Factory func(t *testing.T) Store

// RunStoreContract exercises the durable transcript storage invariants shared by
// all play transcript adapters.
func RunStoreContract(t *testing.T, newStore Factory) {
	t.Helper()

	t.Run("empty session has zero state", func(t *testing.T) {
		t.Parallel()

		store := openStore(t, newStore)
		scope := transcript.Scope{CampaignID: "camp-1", SessionID: "sess-1"}

		latest, err := store.LatestSequence(context.Background(), scope)
		if err != nil {
			t.Fatalf("LatestSequence() error = %v", err)
		}
		if latest != 0 {
			t.Fatalf("LatestSequence() = %d, want 0", latest)
		}

		before, err := store.HistoryBefore(context.Background(), transcript.HistoryBeforeQuery{Scope: scope})
		if err != nil {
			t.Fatalf("HistoryBefore() error = %v", err)
		}
		if len(before) != 0 {
			t.Fatalf("HistoryBefore() = %#v, want empty", before)
		}

		after, err := store.HistoryAfter(context.Background(), transcript.HistoryAfterQuery{Scope: scope})
		if err != nil {
			t.Fatalf("HistoryAfter() error = %v", err)
		}
		if len(after) != 0 {
			t.Fatalf("HistoryAfter() = %#v, want empty", after)
		}
	})

	t.Run("idempotent append and history pagination", func(t *testing.T) {
		t.Parallel()

		store := openStore(t, newStore)
		ctx := context.Background()
		scope := transcript.Scope{CampaignID: "camp-1", SessionID: "sess-1"}

		first, err := store.AppendMessage(ctx, transcript.AppendRequest{
			Scope:           scope,
			Actor:           transcript.MessageActor{ParticipantID: "p1", Name: "Avery"},
			Body:            "hello",
			ClientMessageID: "cli-1",
		})
		if err != nil {
			t.Fatalf("AppendMessage(first) error = %v", err)
		}
		if first.Duplicate {
			t.Fatal("first append unexpectedly marked duplicate")
		}
		if first.Message.SequenceID != 1 {
			t.Fatalf("first sequence = %d, want 1", first.Message.SequenceID)
		}

		duplicate, err := store.AppendMessage(ctx, transcript.AppendRequest{
			Scope:           scope,
			Actor:           transcript.MessageActor{ParticipantID: "p1", Name: "Avery"},
			Body:            "hello",
			ClientMessageID: "cli-1",
		})
		if err != nil {
			t.Fatalf("AppendMessage(duplicate) error = %v", err)
		}
		if !duplicate.Duplicate {
			t.Fatal("duplicate append was not detected")
		}
		if duplicate.Message.MessageID != first.Message.MessageID || duplicate.Message.SequenceID != first.Message.SequenceID {
			t.Fatalf("duplicate result = %#v, want %#v", duplicate.Message, first.Message)
		}

		for i, id := range []string{"cli-2", "cli-3", "cli-4"} {
			result, err := store.AppendMessage(ctx, transcript.AppendRequest{
				Scope:           scope,
				Actor:           transcript.MessageActor{ParticipantID: "p2", Name: "Bo"},
				Body:            "message",
				ClientMessageID: id,
			})
			if err != nil {
				t.Fatalf("AppendMessage(%d) error = %v", i+2, err)
			}
			if result.Message.SequenceID != int64(i+2) {
				t.Fatalf("sequence = %d, want %d", result.Message.SequenceID, i+2)
			}
		}

		latest, err := store.LatestSequence(ctx, scope)
		if err != nil {
			t.Fatalf("LatestSequence() error = %v", err)
		}
		if latest != 4 {
			t.Fatalf("LatestSequence() = %d, want 4", latest)
		}

		after, err := store.HistoryAfter(ctx, transcript.HistoryAfterQuery{
			Scope:           scope,
			AfterSequenceID: 2,
		})
		if err != nil {
			t.Fatalf("HistoryAfter() error = %v", err)
		}
		assertSequenceIDs(t, after, 3, 4)

		before, err := store.HistoryBefore(ctx, transcript.HistoryBeforeQuery{
			Scope:            scope,
			BeforeSequenceID: 4,
			Limit:            2,
		})
		if err != nil {
			t.Fatalf("HistoryBefore() error = %v", err)
		}
		assertSequenceIDs(t, before, 2, 3)
	})

	t.Run("concurrent appends stay gapless", func(t *testing.T) {
		t.Parallel()

		store := openStore(t, newStore)
		ctx := context.Background()
		scope := transcript.Scope{CampaignID: "camp-2", SessionID: "sess-2"}
		const writers = 8

		type appendOutcome struct {
			result transcript.AppendResult
			err    error
		}
		outcomes := make([]appendOutcome, writers)
		var wg sync.WaitGroup
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				outcomes[i].result, outcomes[i].err = store.AppendMessage(ctx, transcript.AppendRequest{
					Scope:           scope,
					Actor:           transcript.MessageActor{ParticipantID: "p1", Name: "Avery"},
					Body:            "message",
					ClientMessageID: clientMessageID(i),
				})
			}(i)
		}
		wg.Wait()

		sequenceIDs := make([]int, 0, writers)
		for i, outcome := range outcomes {
			if outcome.err != nil {
				t.Fatalf("AppendMessage(%d) error = %v", i, outcome.err)
			}
			if outcome.result.Duplicate {
				t.Fatalf("AppendMessage(%d) unexpectedly marked duplicate", i)
			}
			sequenceIDs = append(sequenceIDs, int(outcome.result.Message.SequenceID))
		}
		sort.Ints(sequenceIDs)
		for i, sequenceID := range sequenceIDs {
			if sequenceID != i+1 {
				t.Fatalf("sequence ids = %#v, want contiguous 1..%d", sequenceIDs, writers)
			}
		}

		history, err := store.HistoryAfter(ctx, transcript.HistoryAfterQuery{Scope: scope})
		if err != nil {
			t.Fatalf("HistoryAfter() error = %v", err)
		}
		assertSequenceIDs(t, history, 1, 2, 3, 4, 5, 6, 7, 8)
	})

	t.Run("concurrent duplicate client ids collapse to one message", func(t *testing.T) {
		t.Parallel()

		store := openStore(t, newStore)
		ctx := context.Background()
		request := transcript.AppendRequest{
			Scope:           transcript.Scope{CampaignID: "camp-3", SessionID: "sess-3"},
			Actor:           transcript.MessageActor{ParticipantID: "p1", Name: "Avery"},
			Body:            "same message",
			ClientMessageID: "same-client-id",
		}

		var (
			left  transcript.AppendResult
			right transcript.AppendResult
			errL  error
			errR  error
			wg    sync.WaitGroup
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			left, errL = store.AppendMessage(ctx, request)
		}()
		go func() {
			defer wg.Done()
			right, errR = store.AppendMessage(ctx, request)
		}()
		wg.Wait()

		if errL != nil || errR != nil {
			t.Fatalf("duplicate append errors = %v / %v", errL, errR)
		}
		if left.Message.MessageID != right.Message.MessageID || left.Message.SequenceID != right.Message.SequenceID {
			t.Fatalf("duplicate results diverged: %#v vs %#v", left.Message, right.Message)
		}
		if left.Duplicate == right.Duplicate {
			t.Fatalf("duplicate flags = %v and %v, want one original and one duplicate", left.Duplicate, right.Duplicate)
		}
		history, err := store.HistoryAfter(ctx, transcript.HistoryAfterQuery{Scope: request.Scope})
		if err != nil {
			t.Fatalf("HistoryAfter() error = %v", err)
		}
		assertSequenceIDs(t, history, 1)
	})
}

func openStore(t *testing.T, newStore Factory) Store {
	t.Helper()

	store := newStore(t)
	if closer, ok := store.(interface{ Close() error }); ok {
		t.Cleanup(func() {
			_ = closer.Close()
		})
	}
	return store
}

func assertSequenceIDs(t *testing.T, messages []transcript.Message, want ...int64) {
	t.Helper()

	if len(messages) != len(want) {
		t.Fatalf("message count = %d, want %d (%#v)", len(messages), len(want), messages)
	}
	for i, message := range messages {
		if message.SequenceID != want[i] {
			t.Fatalf("messages[%d].SequenceID = %d, want %d", i, message.SequenceID, want[i])
		}
	}
}

func clientMessageID(index int) string {
	return "cli-" + string(rune('a'+index))
}
