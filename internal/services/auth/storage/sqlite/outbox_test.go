package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
)

func TestIntegrationOutboxEnqueueLeaseAndAckSucceeded(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)

	event := storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "auth.signup_completed",
		PayloadJSON:   `{"user_id":"user-1"}`,
		DedupeKey:     "signup_completed:user:user-1:v1",
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.EnqueueIntegrationOutboxEvent(context.Background(), event); err != nil {
		t.Fatalf("enqueue outbox event: %v", err)
	}

	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 10, now, 5*time.Minute)
	if err != nil {
		t.Fatalf("lease outbox events: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased len = %d, want 1", len(leased))
	}
	if leased[0].ID != event.ID {
		t.Fatalf("leased id = %q, want %q", leased[0].ID, event.ID)
	}
	if leased[0].Status != storage.IntegrationOutboxStatusLeased {
		t.Fatalf("leased status = %q, want %q", leased[0].Status, storage.IntegrationOutboxStatusLeased)
	}
	if leased[0].LeaseOwner != "worker-1" {
		t.Fatalf("lease owner = %q, want %q", leased[0].LeaseOwner, "worker-1")
	}
	if leased[0].LeaseExpiresAt == nil {
		t.Fatal("expected lease expiry")
	}

	// Wrong owner cannot ack.
	if err := store.MarkIntegrationOutboxSucceeded(context.Background(), event.ID, "worker-2", now.Add(time.Minute)); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong owner ack, got %v", err)
	}

	if err := store.MarkIntegrationOutboxSucceeded(context.Background(), event.ID, "worker-1", now.Add(time.Minute)); err != nil {
		t.Fatalf("ack succeeded: %v", err)
	}

	updated, err := store.GetIntegrationOutboxEvent(context.Background(), event.ID)
	if err != nil {
		t.Fatalf("get outbox event: %v", err)
	}
	if updated.Status != storage.IntegrationOutboxStatusSucceeded {
		t.Fatalf("status = %q, want %q", updated.Status, storage.IntegrationOutboxStatusSucceeded)
	}
	if updated.LeaseOwner != "" {
		t.Fatalf("lease owner = %q, want empty", updated.LeaseOwner)
	}
	if updated.LeaseExpiresAt != nil {
		t.Fatalf("lease expiry = %v, want nil", updated.LeaseExpiresAt)
	}
	if updated.ProcessedAt == nil {
		t.Fatal("expected processed_at")
	}
}

func TestIntegrationOutboxLeaseRespectsExpiry(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 22, 5, 0, 0, time.UTC)

	if err := store.EnqueueIntegrationOutboxEvent(context.Background(), storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "auth.signup_completed",
		PayloadJSON:   `{"user_id":"user-1"}`,
		DedupeKey:     "signup_completed:user:user-1:v1",
		Status:        storage.IntegrationOutboxStatusPending,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("enqueue outbox event: %v", err)
	}

	firstLease, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 1, now, 10*time.Minute)
	if err != nil {
		t.Fatalf("lease outbox events: %v", err)
	}
	if len(firstLease) != 1 {
		t.Fatalf("first lease len = %d, want 1", len(firstLease))
	}

	// Not yet expired.
	secondLease, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-2", 1, now.Add(9*time.Minute), 10*time.Minute)
	if err != nil {
		t.Fatalf("second lease: %v", err)
	}
	if len(secondLease) != 0 {
		t.Fatalf("second lease len = %d, want 0", len(secondLease))
	}

	// Expired lease can be reclaimed.
	thirdLease, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-2", 1, now.Add(11*time.Minute), 10*time.Minute)
	if err != nil {
		t.Fatalf("third lease: %v", err)
	}
	if len(thirdLease) != 1 {
		t.Fatalf("third lease len = %d, want 1", len(thirdLease))
	}
	if thirdLease[0].LeaseOwner != "worker-2" {
		t.Fatalf("lease owner = %q, want %q", thirdLease[0].LeaseOwner, "worker-2")
	}
}

func TestIntegrationOutboxRetryAndDead(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 22, 10, 0, 0, time.UTC)

	if err := store.EnqueueIntegrationOutboxEvent(context.Background(), storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "auth.signup_completed",
		PayloadJSON:   `{"user_id":"user-1"}`,
		DedupeKey:     "signup_completed:user:user-1:v1",
		Status:        storage.IntegrationOutboxStatusPending,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("enqueue outbox event: %v", err)
	}

	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 1, now, time.Minute)
	if err != nil {
		t.Fatalf("lease outbox events: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased len = %d, want 1", len(leased))
	}

	retryAt := now.Add(3 * time.Minute)
	if err := store.MarkIntegrationOutboxRetry(context.Background(), "evt-1", "worker-1", retryAt, "temporary failure"); err != nil {
		t.Fatalf("mark retry: %v", err)
	}

	retried, err := store.GetIntegrationOutboxEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("get retried event: %v", err)
	}
	if retried.Status != storage.IntegrationOutboxStatusPending {
		t.Fatalf("status = %q, want %q", retried.Status, storage.IntegrationOutboxStatusPending)
	}
	if retried.AttemptCount != 1 {
		t.Fatalf("attempt count = %d, want 1", retried.AttemptCount)
	}
	if !retried.NextAttemptAt.Equal(retryAt) {
		t.Fatalf("next attempt at = %v, want %v", retried.NextAttemptAt, retryAt)
	}
	if retried.LastError != "temporary failure" {
		t.Fatalf("last error = %q, want %q", retried.LastError, "temporary failure")
	}

	leasedAgain, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 1, retryAt, time.Minute)
	if err != nil {
		t.Fatalf("lease outbox events after retry: %v", err)
	}
	if len(leasedAgain) != 1 {
		t.Fatalf("leased again len = %d, want 1", len(leasedAgain))
	}

	if err := store.MarkIntegrationOutboxDead(context.Background(), "evt-1", "worker-1", "permanent failure", retryAt.Add(time.Minute)); err != nil {
		t.Fatalf("mark dead: %v", err)
	}

	dead, err := store.GetIntegrationOutboxEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("get dead event: %v", err)
	}
	if dead.Status != storage.IntegrationOutboxStatusDead {
		t.Fatalf("status = %q, want %q", dead.Status, storage.IntegrationOutboxStatusDead)
	}
	if dead.AttemptCount != 2 {
		t.Fatalf("attempt count = %d, want 2", dead.AttemptCount)
	}
	if dead.ProcessedAt == nil {
		t.Fatal("expected processed_at on dead event")
	}
}

func TestIntegrationOutboxEnqueueDedupeNoop(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 22, 15, 0, 0, time.UTC)

	first := storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "auth.signup_completed",
		PayloadJSON:   `{"user_id":"user-1"}`,
		DedupeKey:     "signup_completed:user:user-1:v1",
		Status:        storage.IntegrationOutboxStatusPending,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	second := first
	second.ID = "evt-2"
	second.PayloadJSON = `{"user_id":"user-1","duplicate":true}`

	if err := store.EnqueueIntegrationOutboxEvent(context.Background(), first); err != nil {
		t.Fatalf("enqueue first outbox event: %v", err)
	}
	if err := store.EnqueueIntegrationOutboxEvent(context.Background(), second); err != nil {
		t.Fatalf("enqueue second outbox event: %v", err)
	}

	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 10, now, time.Minute)
	if err != nil {
		t.Fatalf("lease outbox events: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased len = %d, want 1", len(leased))
	}
	if leased[0].ID != first.ID {
		t.Fatalf("leased id = %q, want %q", leased[0].ID, first.ID)
	}
}
