package aifakes

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// ProviderGrantStore is an in-memory provider-grant repository fake.
type ProviderGrantStore struct {
	ProviderGrants map[string]providergrant.ProviderGrant
	PutErr         error
	GetErr         error
	ListErr        error
	DeleteErr      error
}

// NewProviderGrantStore creates an initialized provider-grant fake.
func NewProviderGrantStore() *ProviderGrantStore {
	return &ProviderGrantStore{ProviderGrants: make(map[string]providergrant.ProviderGrant)}
}

// PutProviderGrant stores a provider grant.
func (s *ProviderGrantStore) PutProviderGrant(_ context.Context, grant providergrant.ProviderGrant) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.ProviderGrants[grant.ID] = grant
	return nil
}

// GetProviderGrant returns a provider grant by ID.
func (s *ProviderGrantStore) GetProviderGrant(_ context.Context, providerGrantID string) (providergrant.ProviderGrant, error) {
	if s.GetErr != nil {
		return providergrant.ProviderGrant{}, s.GetErr
	}
	grant, ok := s.ProviderGrants[providerGrantID]
	if !ok {
		return providergrant.ProviderGrant{}, storage.ErrNotFound
	}
	return grant, nil
}

// ListProviderGrantsByOwner lists provider grants for an owner.
func (s *ProviderGrantStore) ListProviderGrantsByOwner(_ context.Context, ownerUserID string, _ int, _ string, filter providergrant.Filter) (providergrant.Page, error) {
	if s.ListErr != nil {
		return providergrant.Page{}, s.ListErr
	}
	items := make([]providergrant.ProviderGrant, 0)
	for _, grant := range s.ProviderGrants {
		if grant.OwnerUserID != ownerUserID {
			continue
		}
		if filter.Provider != "" && !strings.EqualFold(string(grant.Provider), string(filter.Provider)) {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(string(grant.Status), string(filter.Status)) {
			continue
		}
		items = append(items, grant)
	}
	return providergrant.Page{ProviderGrants: items}, nil
}
