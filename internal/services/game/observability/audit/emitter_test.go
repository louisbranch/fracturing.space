package audit

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type fakeAuditStore struct {
	last  storage.AuditEvent
	count int
}

func (s *fakeAuditStore) AppendAuditEvent(ctx context.Context, evt storage.AuditEvent) error {
	s.last = evt
	s.count++
	return nil
}

func TestEmitterNoopWhenNil(t *testing.T) {
	var emitter *Emitter
	if err := emitter.Emit(context.Background(), storage.AuditEvent{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEmitterNoopWhenStoreNil(t *testing.T) {
	emitter := &Emitter{}
	if err := emitter.Emit(context.Background(), storage.AuditEvent{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEmitterAddsTimestamp(t *testing.T) {
	store := &fakeAuditStore{}
	clockTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	emitter := &Emitter{store: store, clock: func() time.Time { return clockTime }}

	if err := emitter.Emit(context.Background(), storage.AuditEvent{EventName: "test"}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if store.count != 1 {
		t.Fatalf("expected 1 event, got %d", store.count)
	}
	if store.last.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
	if !store.last.Timestamp.Equal(clockTime) {
		t.Fatalf("expected timestamp %v, got %v", clockTime, store.last.Timestamp)
	}
}

func TestEmitterPreservesTimestamp(t *testing.T) {
	store := &fakeAuditStore{}
	clockTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	setTime := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	emitter := &Emitter{store: store, clock: func() time.Time { return clockTime }}

	if err := emitter.Emit(context.Background(), storage.AuditEvent{EventName: "test", Timestamp: setTime}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if store.count != 1 {
		t.Fatalf("expected 1 event, got %d", store.count)
	}
	if !store.last.Timestamp.Equal(setTime) {
		t.Fatalf("expected timestamp %v, got %v", setTime, store.last.Timestamp)
	}
}

func TestEmitterUsesTimeNowWhenClockNil(t *testing.T) {
	store := &fakeAuditStore{}
	emitter := &Emitter{store: store, clock: nil}

	if err := emitter.Emit(context.Background(), storage.AuditEvent{EventName: "test"}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if store.count != 1 {
		t.Fatalf("expected 1 event, got %d", store.count)
	}
	if store.last.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
}
