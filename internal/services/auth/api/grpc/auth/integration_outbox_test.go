package auth

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestLeaseIntegrationOutboxEvents_Success(t *testing.T) {
	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 21, 22, 30, 0, 0, time.UTC)
	if err := store.EnqueueIntegrationOutboxEvent(context.Background(), storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "auth.signup_completed",
		PayloadJSON:   `{"user_id":"user-1"}`,
		DedupeKey:     "signup_completed:user:user-1:v1",
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("enqueue outbox event: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	resp, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &authv1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   "worker-1",
		Limit:      10,
		LeaseTtlMs: int64((5 * time.Minute) / time.Millisecond),
		Now:        timestamppb.New(now),
	})
	if err != nil {
		t.Fatalf("lease integration outbox events: %v", err)
	}
	if len(resp.GetEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.GetEvents()))
	}
	if resp.GetEvents()[0].GetId() != "evt-1" {
		t.Fatalf("event id = %q, want %q", resp.GetEvents()[0].GetId(), "evt-1")
	}
	if resp.GetEvents()[0].GetStatus() != storage.IntegrationOutboxStatusLeased {
		t.Fatalf("event status = %q, want %q", resp.GetEvents()[0].GetStatus(), storage.IntegrationOutboxStatusLeased)
	}
}

func TestAckIntegrationOutboxEvent_RetryAndSucceeded(t *testing.T) {
	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 21, 22, 35, 0, 0, time.UTC)
	if err := store.EnqueueIntegrationOutboxEvent(context.Background(), storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "auth.signup_completed",
		PayloadJSON:   `{"user_id":"user-1"}`,
		DedupeKey:     "signup_completed:user:user-1:v1",
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("enqueue outbox event: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	leaseResp, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &authv1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   "worker-1",
		Limit:      1,
		LeaseTtlMs: int64((time.Minute) / time.Millisecond),
		Now:        timestamppb.New(now),
	})
	if err != nil {
		t.Fatalf("lease integration outbox events: %v", err)
	}
	if len(leaseResp.GetEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(leaseResp.GetEvents()))
	}

	retryAt := now.Add(2 * time.Minute)
	if _, err := svc.AckIntegrationOutboxEvent(context.Background(), &authv1.AckIntegrationOutboxEventRequest{
		EventId:       "evt-1",
		Consumer:      "worker-1",
		Outcome:       authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY,
		NextAttemptAt: timestamppb.New(retryAt),
		LastError:     "temporary failure",
	}); err != nil {
		t.Fatalf("ack retry: %v", err)
	}

	retried, err := store.GetIntegrationOutboxEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("get retried event: %v", err)
	}
	if retried.Status != storage.IntegrationOutboxStatusPending {
		t.Fatalf("status after retry = %q, want %q", retried.Status, storage.IntegrationOutboxStatusPending)
	}
	if retried.AttemptCount != 1 {
		t.Fatalf("attempt count after retry = %d, want 1", retried.AttemptCount)
	}

	leaseResp, err = svc.LeaseIntegrationOutboxEvents(context.Background(), &authv1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   "worker-1",
		Limit:      1,
		LeaseTtlMs: int64((time.Minute) / time.Millisecond),
		Now:        timestamppb.New(retryAt),
	})
	if err != nil {
		t.Fatalf("lease integration outbox events second time: %v", err)
	}
	if len(leaseResp.GetEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(leaseResp.GetEvents()))
	}

	if _, err := svc.AckIntegrationOutboxEvent(context.Background(), &authv1.AckIntegrationOutboxEventRequest{
		EventId:     "evt-1",
		Consumer:    "worker-1",
		Outcome:     authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED,
		ProcessedAt: timestamppb.New(retryAt.Add(time.Minute)),
	}); err != nil {
		t.Fatalf("ack succeeded: %v", err)
	}

	done, err := store.GetIntegrationOutboxEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("get succeeded event: %v", err)
	}
	if done.Status != storage.IntegrationOutboxStatusSucceeded {
		t.Fatalf("status after success = %q, want %q", done.Status, storage.IntegrationOutboxStatusSucceeded)
	}
}

func TestAckIntegrationOutboxEvent_RejectsUnspecifiedOutcome(t *testing.T) {
	store := openTempAuthStore(t)
	svc := NewAuthService(store, store, nil)

	_, err := svc.AckIntegrationOutboxEvent(context.Background(), &authv1.AckIntegrationOutboxEventRequest{
		EventId:  "evt-1",
		Consumer: "worker-1",
		Outcome:  authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_UNSPECIFIED,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}
