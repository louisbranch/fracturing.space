package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (s *fakeContentStore) PutDaggerheartDamageType(_ context.Context, d contentstore.DaggerheartDamageTypeEntry) error {
	s.damageTypes[d.ID] = d
	return nil
}

func (s *fakeContentStore) GetDaggerheartDamageType(_ context.Context, id string) (contentstore.DaggerheartDamageTypeEntry, error) {
	d, ok := s.damageTypes[id]
	if !ok {
		return contentstore.DaggerheartDamageTypeEntry{}, storage.ErrNotFound
	}
	return d, nil
}

func (s *fakeContentStore) ListDaggerheartDamageTypes(_ context.Context) ([]contentstore.DaggerheartDamageTypeEntry, error) {
	result := make([]contentstore.DaggerheartDamageTypeEntry, 0, len(s.damageTypes))
	for _, d := range s.damageTypes {
		result = append(result, d)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartDamageType(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartDomain(_ context.Context, d contentstore.DaggerheartDomain) error {
	s.domains[d.ID] = d
	return nil
}

func (s *fakeContentStore) GetDaggerheartDomain(_ context.Context, id string) (contentstore.DaggerheartDomain, error) {
	d, ok := s.domains[id]
	if !ok {
		return contentstore.DaggerheartDomain{}, storage.ErrNotFound
	}
	return d, nil
}

func (s *fakeContentStore) ListDaggerheartDomains(_ context.Context) ([]contentstore.DaggerheartDomain, error) {
	result := make([]contentstore.DaggerheartDomain, 0, len(s.domains))
	for _, d := range s.domains {
		result = append(result, d)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartDomain(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartDomainCard(_ context.Context, c contentstore.DaggerheartDomainCard) error {
	s.domainCards[c.ID] = c
	return nil
}

func (s *fakeContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (contentstore.DaggerheartDomainCard, error) {
	c, ok := s.domainCards[id]
	if !ok {
		return contentstore.DaggerheartDomainCard{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *fakeContentStore) ListDaggerheartDomainCards(_ context.Context) ([]contentstore.DaggerheartDomainCard, error) {
	result := make([]contentstore.DaggerheartDomainCard, 0, len(s.domainCards))
	for _, c := range s.domainCards {
		result = append(result, c)
	}
	return result, nil
}

func (s *fakeContentStore) ListDaggerheartDomainCardsByDomain(_ context.Context, domainID string) ([]contentstore.DaggerheartDomainCard, error) {
	var result []contentstore.DaggerheartDomainCard
	for _, c := range s.domainCards {
		if c.DomainID == domainID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartDomainCard(_ context.Context, _ string) error { return nil }
