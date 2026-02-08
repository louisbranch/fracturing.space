package storage

import (
	"context"
	"time"
)

// UserSessionStore persists admin user session records.
type UserSessionStore interface {
	PutUserSession(ctx context.Context, sessionID string, createdAt time.Time) error
}

// Store is a composite interface for admin storage concerns.
type Store interface {
	UserSessionStore
	Close() error
}
