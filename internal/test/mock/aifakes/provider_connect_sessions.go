package aifakes

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/ai/providerconnect"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// ProviderConnectSessionStore is an in-memory provider-connect-session fake.
type ProviderConnectSessionStore struct {
	ConnectSessions map[string]providerconnect.Session
	PutErr          error
	GetErr          error
	DeleteErr       error
}

// NewProviderConnectSessionStore creates an initialized connect-session fake.
func NewProviderConnectSessionStore() *ProviderConnectSessionStore {
	return &ProviderConnectSessionStore{ConnectSessions: make(map[string]providerconnect.Session)}
}

// PutProviderConnectSession stores a provider connect session.
func (s *ProviderConnectSessionStore) PutProviderConnectSession(_ context.Context, session providerconnect.Session) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.ConnectSessions[session.ID] = session
	return nil
}

// GetProviderConnectSession returns a provider connect session by ID.
func (s *ProviderConnectSessionStore) GetProviderConnectSession(_ context.Context, connectSessionID string) (providerconnect.Session, error) {
	if s.GetErr != nil {
		return providerconnect.Session{}, s.GetErr
	}
	rec, ok := s.ConnectSessions[connectSessionID]
	if !ok {
		return providerconnect.Session{}, storage.ErrNotFound
	}
	return rec, nil
}

// CompleteProviderConnectSession marks a pending connect session completed.
func (s *ProviderConnectSessionStore) CompleteProviderConnectSession(_ context.Context, session providerconnect.Session) error {
	if s.DeleteErr != nil {
		return s.DeleteErr
	}
	rec, ok := s.ConnectSessions[session.ID]
	if !ok || rec.OwnerUserID != session.OwnerUserID || rec.Status != providerconnect.StatusPending || session.Status != providerconnect.StatusCompleted {
		return storage.ErrNotFound
	}
	rec.Status = session.Status
	rec.CompletedAt = session.CompletedAt
	rec.UpdatedAt = session.UpdatedAt
	s.ConnectSessions[session.ID] = rec
	return nil
}
