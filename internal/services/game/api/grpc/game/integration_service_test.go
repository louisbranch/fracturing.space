package game

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestLeaseIntegrationOutboxEvents_Success(t *testing.T) {
	store := newFakeIntegrationOutboxEventStore()
	now := time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC)
	store.outbox["evt-1"] = storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "game.invite.created.v1",
		PayloadJSON:   `{"invite_id":"invite-1","campaign_id":"campaign-1","recipient_user_id":"user-2"}`,
		DedupeKey:     "invite:invite-1:created",
		Status:        storage.IntegrationOutboxStatusPending,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	svc := NewIntegrationService(store)
	resp, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &gamev1.LeaseIntegrationOutboxEventsRequest{
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

type fakeIntegrationOutboxEventStore struct {
	*gametest.FakeEventStore
	outbox map[string]storage.IntegrationOutboxEvent
}

func newFakeIntegrationOutboxEventStore() *fakeIntegrationOutboxEventStore {
	return &fakeIntegrationOutboxEventStore{
		FakeEventStore: gametest.NewFakeEventStore(),
		outbox:         make(map[string]storage.IntegrationOutboxEvent),
	}
}

func (s *fakeIntegrationOutboxEventStore) EnqueueIntegrationOutboxEvent(_ context.Context, event storage.IntegrationOutboxEvent) error {
	s.outbox[event.ID] = event
	return nil
}

func (s *fakeIntegrationOutboxEventStore) GetIntegrationOutboxEvent(_ context.Context, id string) (storage.IntegrationOutboxEvent, error) {
	event, ok := s.outbox[id]
	if !ok {
		return storage.IntegrationOutboxEvent{}, storage.ErrNotFound
	}
	return event, nil
}

func (s *fakeIntegrationOutboxEventStore) LeaseIntegrationOutboxEvents(_ context.Context, consumer string, limit int, now time.Time, leaseTTL time.Duration) ([]storage.IntegrationOutboxEvent, error) {
	results := make([]storage.IntegrationOutboxEvent, 0, limit)
	for id, event := range s.outbox {
		if limit > 0 && len(results) >= limit {
			break
		}
		if event.Status == storage.IntegrationOutboxStatusPending && !event.NextAttemptAt.After(now) {
			event.Status = storage.IntegrationOutboxStatusLeased
			event.LeaseOwner = consumer
			expiresAt := now.Add(leaseTTL)
			event.LeaseExpiresAt = &expiresAt
			event.UpdatedAt = now
			s.outbox[id] = event
			results = append(results, event)
		}
	}
	return results, nil
}

func (s *fakeIntegrationOutboxEventStore) MarkIntegrationOutboxSucceeded(_ context.Context, id string, consumer string, processedAt time.Time) error {
	event, ok := s.outbox[id]
	if !ok || event.LeaseOwner != consumer {
		return storage.ErrNotFound
	}
	event.Status = storage.IntegrationOutboxStatusSucceeded
	event.LeaseOwner = ""
	event.LeaseExpiresAt = nil
	event.ProcessedAt = &processedAt
	event.UpdatedAt = processedAt
	s.outbox[id] = event
	return nil
}

func (s *fakeIntegrationOutboxEventStore) MarkIntegrationOutboxRetry(_ context.Context, id string, consumer string, nextAttemptAt time.Time, lastError string) error {
	event, ok := s.outbox[id]
	if !ok || event.LeaseOwner != consumer {
		return storage.ErrNotFound
	}
	event.Status = storage.IntegrationOutboxStatusPending
	event.AttemptCount++
	event.LeaseOwner = ""
	event.LeaseExpiresAt = nil
	event.NextAttemptAt = nextAttemptAt
	event.LastError = lastError
	event.UpdatedAt = nextAttemptAt
	s.outbox[id] = event
	return nil
}

func (s *fakeIntegrationOutboxEventStore) MarkIntegrationOutboxDead(_ context.Context, id string, consumer string, lastError string, processedAt time.Time) error {
	event, ok := s.outbox[id]
	if !ok || event.LeaseOwner != consumer {
		return storage.ErrNotFound
	}
	event.Status = storage.IntegrationOutboxStatusDead
	event.AttemptCount++
	event.LeaseOwner = ""
	event.LeaseExpiresAt = nil
	event.LastError = lastError
	event.ProcessedAt = &processedAt
	event.UpdatedAt = processedAt
	s.outbox[id] = event
	return nil
}
