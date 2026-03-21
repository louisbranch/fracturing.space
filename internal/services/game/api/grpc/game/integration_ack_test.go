package game

import (
	"context"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAckIntegrationOutboxEvent_RetryAndSucceeded(t *testing.T) {
	store := newFakeIntegrationOutboxEventStore()
	now := time.Date(2026, 3, 9, 14, 5, 0, 0, time.UTC)
	store.outbox["evt-1"] = storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "game.invite.claimed.v1",
		PayloadJSON:   `{"invite_id":"invite-1","campaign_id":"campaign-1","recipient_user_id":"user-2"}`,
		DedupeKey:     "invite:invite-1:accepted",
		Status:        storage.IntegrationOutboxStatusPending,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	svc := NewIntegrationService(store)
	leaseResp, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &gamev1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   "worker-1",
		Limit:      1,
		LeaseTtlMs: int64(time.Minute / time.Millisecond),
		Now:        timestamppb.New(now),
	})
	if err != nil {
		t.Fatalf("lease integration outbox events: %v", err)
	}
	if len(leaseResp.GetEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(leaseResp.GetEvents()))
	}

	retryAt := now.Add(2 * time.Minute)
	if _, err := svc.AckIntegrationOutboxEvent(context.Background(), &gamev1.AckIntegrationOutboxEventRequest{
		EventId:       "evt-1",
		Consumer:      "worker-1",
		Outcome:       gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY,
		NextAttemptAt: timestamppb.New(retryAt),
		LastError:     "temporary failure",
	}); err != nil {
		t.Fatalf("ack retry: %v", err)
	}
	if store.outbox["evt-1"].Status != storage.IntegrationOutboxStatusPending {
		t.Fatalf("status after retry = %q, want %q", store.outbox["evt-1"].Status, storage.IntegrationOutboxStatusPending)
	}

	if _, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &gamev1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   "worker-1",
		Limit:      1,
		LeaseTtlMs: int64(time.Minute / time.Millisecond),
		Now:        timestamppb.New(retryAt),
	}); err != nil {
		t.Fatalf("lease integration outbox events second time: %v", err)
	}
	if _, err := svc.AckIntegrationOutboxEvent(context.Background(), &gamev1.AckIntegrationOutboxEventRequest{
		EventId:  "evt-1",
		Consumer: "worker-1",
		Outcome:  gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED,
	}); err != nil {
		t.Fatalf("ack succeeded: %v", err)
	}
	if store.outbox["evt-1"].Status != storage.IntegrationOutboxStatusSucceeded {
		t.Fatalf("status after success = %q, want %q", store.outbox["evt-1"].Status, storage.IntegrationOutboxStatusSucceeded)
	}
}

func TestAckIntegrationOutboxEvent_RejectsUnspecifiedOutcome(t *testing.T) {
	svc := NewIntegrationService(newFakeIntegrationOutboxEventStore())

	_, err := svc.AckIntegrationOutboxEvent(context.Background(), &gamev1.AckIntegrationOutboxEventRequest{
		EventId:  "evt-1",
		Consumer: "worker-1",
		Outcome:  gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_UNSPECIFIED,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}
