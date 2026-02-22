package storage

import (
	"context"
	"time"
)

// AttemptRecord is one durable worker processing outcome record.
type AttemptRecord struct {
	ID           int64
	EventID      string
	EventType    string
	Consumer     string
	Outcome      string
	AttemptCount int32
	LastError    string
	CreatedAt    time.Time
}

// AttemptStore persists worker processing attempt records.
type AttemptStore interface {
	RecordAttempt(ctx context.Context, attempt AttemptRecord) error
	ListAttempts(ctx context.Context, limit int) ([]AttemptRecord, error)
}
