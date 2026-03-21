package aifakes

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// CredentialStore is an in-memory credential repository fake.
type CredentialStore struct {
	Credentials map[string]storage.CredentialRecord
}

// NewCredentialStore creates an initialized credential fake.
func NewCredentialStore() *CredentialStore {
	return &CredentialStore{Credentials: make(map[string]storage.CredentialRecord)}
}

// PutCredential stores a credential record.
func (s *CredentialStore) PutCredential(_ context.Context, record storage.CredentialRecord) error {
	for id, existing := range s.Credentials {
		if id == record.ID {
			continue
		}
		if !sameNormalizedOwner(record.OwnerUserID, existing.OwnerUserID) {
			continue
		}
		if existing.RevokedAt != nil || strings.EqualFold(strings.TrimSpace(existing.Status), "revoked") {
			continue
		}
		if normalizedLabel(record.Label) == normalizedLabel(existing.Label) {
			return storage.ErrConflict
		}
	}
	s.Credentials[record.ID] = record
	return nil
}

// GetCredential returns a credential by ID.
func (s *CredentialStore) GetCredential(_ context.Context, credentialID string) (storage.CredentialRecord, error) {
	rec, ok := s.Credentials[credentialID]
	if !ok {
		return storage.CredentialRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// ListCredentialsByOwner lists credentials for an owner.
func (s *CredentialStore) ListCredentialsByOwner(_ context.Context, ownerUserID string, _ int, _ string) (storage.CredentialPage, error) {
	items := make([]storage.CredentialRecord, 0)
	for _, rec := range s.Credentials {
		if rec.OwnerUserID == ownerUserID {
			items = append(items, rec)
		}
	}
	return storage.CredentialPage{Credentials: items}, nil
}
