package aifakes

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// ProviderGrantStore is an in-memory provider-grant repository fake.
type ProviderGrantStore struct {
	ProviderGrants map[string]storage.ProviderGrantRecord
}

// NewProviderGrantStore creates an initialized provider-grant fake.
func NewProviderGrantStore() *ProviderGrantStore {
	return &ProviderGrantStore{ProviderGrants: make(map[string]storage.ProviderGrantRecord)}
}

// PutProviderGrant stores a provider grant record.
func (s *ProviderGrantStore) PutProviderGrant(_ context.Context, record storage.ProviderGrantRecord) error {
	s.ProviderGrants[record.ID] = record
	return nil
}

// GetProviderGrant returns a provider grant by ID.
func (s *ProviderGrantStore) GetProviderGrant(_ context.Context, providerGrantID string) (storage.ProviderGrantRecord, error) {
	rec, ok := s.ProviderGrants[providerGrantID]
	if !ok {
		return storage.ProviderGrantRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// ListProviderGrantsByOwner lists provider grants for an owner.
func (s *ProviderGrantStore) ListProviderGrantsByOwner(_ context.Context, ownerUserID string, _ int, _ string, filter storage.ProviderGrantFilter) (storage.ProviderGrantPage, error) {
	providerID := strings.ToLower(strings.TrimSpace(filter.Provider))
	status := strings.ToLower(strings.TrimSpace(filter.Status))
	items := make([]storage.ProviderGrantRecord, 0)
	for _, rec := range s.ProviderGrants {
		if rec.OwnerUserID != ownerUserID {
			continue
		}
		if providerID != "" && !strings.EqualFold(strings.TrimSpace(rec.Provider), providerID) {
			continue
		}
		if status != "" && !strings.EqualFold(strings.TrimSpace(rec.Status), status) {
			continue
		}
		items = append(items, rec)
	}
	return storage.ProviderGrantPage{ProviderGrants: items}, nil
}
