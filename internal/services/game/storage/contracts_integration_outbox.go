package storage

import (
	"context"
	"time"
)

const (
	// IntegrationOutboxStatusPending is ready for leasing and processing.
	IntegrationOutboxStatusPending = "pending"
	// IntegrationOutboxStatusLeased is currently leased by one worker.
	IntegrationOutboxStatusLeased = "leased"
	// IntegrationOutboxStatusSucceeded finished successfully.
	IntegrationOutboxStatusSucceeded = "succeeded"
	// IntegrationOutboxStatusDead exhausted retries and needs operator action.
	IntegrationOutboxStatusDead = "dead"
)

// IntegrationOutboxEvent is one durable game-owned integration work item.
type IntegrationOutboxEvent struct {
	ID             string
	EventType      string
	PayloadJSON    string
	DedupeKey      string
	Status         string
	AttemptCount   int
	NextAttemptAt  time.Time
	LeaseOwner     string
	LeaseExpiresAt *time.Time
	LastError      string
	ProcessedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IntegrationOutboxStore persists durable game integration work for workers.
type IntegrationOutboxStore interface {
	EnqueueIntegrationOutboxEvent(ctx context.Context, event IntegrationOutboxEvent) error
	GetIntegrationOutboxEvent(ctx context.Context, id string) (IntegrationOutboxEvent, error)
	LeaseIntegrationOutboxEvents(ctx context.Context, consumer string, limit int, now time.Time, leaseTTL time.Duration) ([]IntegrationOutboxEvent, error)
	MarkIntegrationOutboxSucceeded(ctx context.Context, id string, consumer string, processedAt time.Time) error
	MarkIntegrationOutboxRetry(ctx context.Context, id string, consumer string, nextAttemptAt time.Time, lastError string) error
	MarkIntegrationOutboxDead(ctx context.Context, id string, consumer string, lastError string, processedAt time.Time) error
}
