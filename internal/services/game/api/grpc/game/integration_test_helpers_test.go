package game

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

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
