package server

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
)

type fakePendingEmailDeliveryStore struct {
	pending []storage.DeliveryRecord
	err     error
	calls   int
}

func (s *fakePendingEmailDeliveryStore) ListPendingDeliveries(_ context.Context, channel storage.DeliveryChannel, limit int, now time.Time) ([]storage.DeliveryRecord, error) {
	s.calls++
	if channel != storage.DeliveryChannelEmail {
		return nil, errors.New("unexpected channel")
	}
	if limit != emailDeliveryBatchLimit {
		return nil, errors.New("unexpected limit")
	}
	if now.IsZero() {
		return nil, errors.New("expected non-zero now")
	}
	if s.err != nil {
		return nil, s.err
	}
	return append([]storage.DeliveryRecord(nil), s.pending...), nil
}

func TestEmailDeliveryWorkerObserveReportsWrappedErrors(t *testing.T) {
	store := &fakePendingEmailDeliveryStore{err: errors.New("db unavailable")}
	worker := newEmailDeliveryWorker(store, time.Second, func() time.Time { return time.Date(2026, 3, 7, 1, 0, 0, 0, time.UTC) }, nil)

	err := worker.Observe(context.Background())
	if err == nil || !strings.Contains(err.Error(), "list pending email deliveries") {
		t.Fatalf("Observe error = %v, want wrapped list error", err)
	}
	if store.calls != 1 {
		t.Fatalf("store calls = %d, want 1", store.calls)
	}
}

func TestEmailDeliveryWorkerObserveLogsWhenPendingExists(t *testing.T) {
	store := &fakePendingEmailDeliveryStore{pending: []storage.DeliveryRecord{{NotificationID: "notif-1", Channel: storage.DeliveryChannelEmail}}}
	logs := make([]string, 0, 1)
	worker := newEmailDeliveryWorker(store, time.Second, func() time.Time { return time.Date(2026, 3, 7, 1, 0, 0, 0, time.UTC) }, func(format string, args ...any) {
		logs = append(logs, format)
	})

	if err := worker.Observe(context.Background()); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if store.calls != 1 {
		t.Fatalf("store calls = %d, want 1", store.calls)
	}
	if len(logs) != 1 || !strings.Contains(logs[0], "observed %d pending deliveries") {
		t.Fatalf("logs = %v, want pending delivery observation", logs)
	}
}

func TestEmailDeliveryWorkerRunStopsOnCancellation(t *testing.T) {
	store := &fakePendingEmailDeliveryStore{}
	worker := newEmailDeliveryWorker(store, 10*time.Millisecond, time.Now, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancellation")
	}
	if store.calls == 0 {
		t.Fatal("expected Run to call Observe at least once")
	}
}
