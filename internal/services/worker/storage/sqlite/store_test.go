package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/worker/storage"
)

func TestRecordAndListAttempts(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 23, 30, 0, 0, time.UTC)

	if err := store.RecordAttempt(context.Background(), storage.AttemptRecord{
		EventID:      "evt-1",
		EventType:    "auth.signup_completed",
		Consumer:     "worker-1",
		Outcome:      "retry",
		AttemptCount: 1,
		LastError:    "temporary error",
		CreatedAt:    now,
	}); err != nil {
		t.Fatalf("record attempt: %v", err)
	}
	if err := store.RecordAttempt(context.Background(), storage.AttemptRecord{
		EventID:      "evt-1",
		EventType:    "auth.signup_completed",
		Consumer:     "worker-1",
		Outcome:      "succeeded",
		AttemptCount: 2,
		CreatedAt:    now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("record attempt second: %v", err)
	}

	attempts, err := store.ListAttempts(context.Background(), 10)
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts len = %d, want 2", len(attempts))
	}
	if attempts[0].Outcome != "succeeded" {
		t.Fatalf("attempts[0].outcome = %q, want %q", attempts[0].Outcome, "succeeded")
	}
	if attempts[1].Outcome != "retry" {
		t.Fatalf("attempts[1].outcome = %q, want %q", attempts[1].Outcome, "retry")
	}
}

func TestRecordAttemptValidation(t *testing.T) {
	store := openTempStore(t)

	if err := store.RecordAttempt(context.Background(), storage.AttemptRecord{}); err == nil {
		t.Fatal("expected validation error for empty attempt")
	}
}

func openTempStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "worker.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}
