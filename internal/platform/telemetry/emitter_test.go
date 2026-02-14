package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type fakeTelemetryStore struct {
	last  storage.TelemetryEvent
	count int
}

func (s *fakeTelemetryStore) AppendTelemetryEvent(ctx context.Context, evt storage.TelemetryEvent) error {
	s.last = evt
	s.count++
	return nil
}

func TestEmitterNoopWhenNil(t *testing.T) {
	var emitter *Emitter
	if err := emitter.Emit(context.Background(), storage.TelemetryEvent{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEmitterNoopWhenStoreNil(t *testing.T) {
	emitter := &Emitter{}
	if err := emitter.Emit(context.Background(), storage.TelemetryEvent{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEmitterAddsTimestamp(t *testing.T) {
	store := &fakeTelemetryStore{}
	clockTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	emitter := &Emitter{store: store, clock: func() time.Time { return clockTime }}

	if err := emitter.Emit(context.Background(), storage.TelemetryEvent{EventName: "test"}); err != nil {
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
	store := &fakeTelemetryStore{}
	clockTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	setTime := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	emitter := &Emitter{store: store, clock: func() time.Time { return clockTime }}

	if err := emitter.Emit(context.Background(), storage.TelemetryEvent{EventName: "test", Timestamp: setTime}); err != nil {
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
	store := &fakeTelemetryStore{}
	emitter := &Emitter{store: store, clock: nil}

	if err := emitter.Emit(context.Background(), storage.TelemetryEvent{EventName: "test"}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if store.count != 1 {
		t.Fatalf("expected 1 event, got %d", store.count)
	}
	if store.last.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
}
