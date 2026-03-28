package aifakes

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// CredentialStore is an in-memory credential repository fake.
type CredentialStore struct {
	Credentials map[string]credential.Credential
	PutErr      error
	GetErr      error
	ListErr     error
}

// NewCredentialStore creates an initialized credential fake.
func NewCredentialStore() *CredentialStore {
	return &CredentialStore{Credentials: make(map[string]credential.Credential)}
}

// PutCredential stores a credential.
func (s *CredentialStore) PutCredential(_ context.Context, c credential.Credential) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	for id, existing := range s.Credentials {
		if id == c.ID {
			continue
		}
		if !sameNormalizedOwner(c.OwnerUserID, existing.OwnerUserID) {
			continue
		}
		if existing.RevokedAt != nil || existing.Status.IsRevoked() {
			continue
		}
		if normalizedLabel(c.Label) == normalizedLabel(existing.Label) {
			return storage.ErrConflict
		}
	}
	s.Credentials[c.ID] = c
	return nil
}

// GetCredential returns a credential by ID.
func (s *CredentialStore) GetCredential(_ context.Context, credentialID string) (credential.Credential, error) {
	if s.GetErr != nil {
		return credential.Credential{}, s.GetErr
	}
	c, ok := s.Credentials[credentialID]
	if !ok {
		return credential.Credential{}, storage.ErrNotFound
	}
	return c, nil
}

// ListCredentialsByOwner lists credentials for an owner.
func (s *CredentialStore) ListCredentialsByOwner(_ context.Context, ownerUserID string, _ int, _ string) (credential.Page, error) {
	if s.ListErr != nil {
		return credential.Page{}, s.ListErr
	}
	items := make([]credential.Credential, 0)
	for _, c := range s.Credentials {
		if c.OwnerUserID == ownerUserID {
			items = append(items, c)
		}
	}
	return credential.Page{Credentials: items}, nil
}
