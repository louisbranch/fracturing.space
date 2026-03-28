package storage

import (
	"context"
	"time"
)

// IntegrationOutboxStatus represents the lifecycle state of an outbox entry.
type IntegrationOutboxStatus = string

const (
	// IntegrationOutboxStatusPending is ready for leasing and processing.
	IntegrationOutboxStatusPending IntegrationOutboxStatus = "pending"
	// IntegrationOutboxStatusLeased is currently leased by one worker.
	IntegrationOutboxStatusLeased IntegrationOutboxStatus = "leased"
	// IntegrationOutboxStatusSucceeded finished successfully.
	IntegrationOutboxStatusSucceeded IntegrationOutboxStatus = "succeeded"
	// IntegrationOutboxStatusDead exhausted retries and needs operator action.
	IntegrationOutboxStatusDead IntegrationOutboxStatus = "dead"
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

// IntegrationOutboxWriter persists new durable game integration work items.
type IntegrationOutboxWriter interface {
	EnqueueIntegrationOutboxEvent(ctx context.Context, event IntegrationOutboxEvent) error
}

// IntegrationOutboxReader loads persisted game integration work by id.
type IntegrationOutboxReader interface {
	GetIntegrationOutboxEvent(ctx context.Context, id string) (IntegrationOutboxEvent, error)
}

// IntegrationOutboxLeaser claims due integration work for one worker.
type IntegrationOutboxLeaser interface {
	LeaseIntegrationOutboxEvents(ctx context.Context, consumer string, limit int, now time.Time, leaseTTL time.Duration) ([]IntegrationOutboxEvent, error)
}

// IntegrationOutboxAcknowledger records worker outcomes for leased work items.
type IntegrationOutboxAcknowledger interface {
	MarkIntegrationOutboxSucceeded(ctx context.Context, id string, consumer string, processedAt time.Time) error
	MarkIntegrationOutboxRetry(ctx context.Context, id string, consumer string, nextAttemptAt time.Time, lastError string) error
	MarkIntegrationOutboxDead(ctx context.Context, id string, consumer string, lastError string, processedAt time.Time) error
}

// IntegrationOutboxWorkerStore is the worker-facing lease/ack surface.
type IntegrationOutboxWorkerStore interface {
	IntegrationOutboxLeaser
	IntegrationOutboxAcknowledger
}

// IntegrationOutboxStore persists durable game integration work for both
// production writers and worker consumers.
type IntegrationOutboxStore interface {
	IntegrationOutboxWriter
	IntegrationOutboxReader
	IntegrationOutboxWorkerStore
}
