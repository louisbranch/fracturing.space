package aifakes

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// ProviderConnectSessionStore is an in-memory provider-connect-session fake.
type ProviderConnectSessionStore struct {
	ConnectSessions map[string]storage.ProviderConnectSessionRecord
}

// NewProviderConnectSessionStore creates an initialized connect-session fake.
func NewProviderConnectSessionStore() *ProviderConnectSessionStore {
	return &ProviderConnectSessionStore{ConnectSessions: make(map[string]storage.ProviderConnectSessionRecord)}
}

// PutProviderConnectSession stores a provider connect session.
func (s *ProviderConnectSessionStore) PutProviderConnectSession(_ context.Context, record storage.ProviderConnectSessionRecord) error {
	s.ConnectSessions[record.ID] = record
	return nil
}

// GetProviderConnectSession returns a provider connect session by ID.
func (s *ProviderConnectSessionStore) GetProviderConnectSession(_ context.Context, connectSessionID string) (storage.ProviderConnectSessionRecord, error) {
	rec, ok := s.ConnectSessions[connectSessionID]
	if !ok {
		return storage.ProviderConnectSessionRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// CompleteProviderConnectSession marks a pending connect session completed.
func (s *ProviderConnectSessionStore) CompleteProviderConnectSession(_ context.Context, ownerUserID string, connectSessionID string, completedAt time.Time) error {
	rec, ok := s.ConnectSessions[connectSessionID]
	if !ok || rec.OwnerUserID != ownerUserID || rec.Status != "pending" {
		return storage.ErrNotFound
	}
	rec.Status = "completed"
	rec.CompletedAt = &completedAt
	rec.UpdatedAt = completedAt
	s.ConnectSessions[connectSessionID] = rec
	return nil
}
