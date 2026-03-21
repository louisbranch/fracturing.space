package audit

import (
	"context"
	"errors"
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

func TestEmitterNoopWhenPolicyDisabled(t *testing.T) {
	emitter := NewEmitter(DisabledPolicy())
	if err := emitter.Emit(context.Background(), storage.AuditEvent{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEmitterEnabledWithoutStoreFails(t *testing.T) {
	emitter := NewEmitter(EnabledPolicy(nil))
	err := emitter.Emit(context.Background(), storage.AuditEvent{})
	if !errors.Is(err, errEnabledAuditStoreRequired) {
		t.Fatalf("emit error = %v, want %v", err, errEnabledAuditStoreRequired)
	}
}

func TestEmitterAddsTimestamp(t *testing.T) {
	store := &fakeAuditStore{}
	clockTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	emitter := &Emitter{policy: EnabledPolicy(store), clock: func() time.Time { return clockTime }}

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
	emitter := &Emitter{policy: EnabledPolicy(store), clock: func() time.Time { return clockTime }}

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
	emitter := &Emitter{policy: EnabledPolicy(store), clock: nil}

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
