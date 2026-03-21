package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (s *fakeContentStore) PutDaggerheartAdversaryEntry(_ context.Context, a contentstore.DaggerheartAdversaryEntry) error {
	s.adversaryEntries[a.ID] = a
	return nil
}

func (s *fakeContentStore) GetDaggerheartAdversaryEntry(_ context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error) {
	a, ok := s.adversaryEntries[id]
	if !ok {
		return contentstore.DaggerheartAdversaryEntry{}, storage.ErrNotFound
	}
	return a, nil
}

func (s *fakeContentStore) ListDaggerheartAdversaryEntries(_ context.Context) ([]contentstore.DaggerheartAdversaryEntry, error) {
	result := make([]contentstore.DaggerheartAdversaryEntry, 0, len(s.adversaryEntries))
	for _, a := range s.adversaryEntries {
		result = append(result, a)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartAdversaryEntry(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartBeastform(_ context.Context, b contentstore.DaggerheartBeastformEntry) error {
	s.beastforms[b.ID] = b
	return nil
}

func (s *fakeContentStore) GetDaggerheartBeastform(_ context.Context, id string) (contentstore.DaggerheartBeastformEntry, error) {
	b, ok := s.beastforms[id]
	if !ok {
		return contentstore.DaggerheartBeastformEntry{}, storage.ErrNotFound
	}
	return b, nil
}

func (s *fakeContentStore) ListDaggerheartBeastforms(_ context.Context) ([]contentstore.DaggerheartBeastformEntry, error) {
	result := make([]contentstore.DaggerheartBeastformEntry, 0, len(s.beastforms))
	for _, b := range s.beastforms {
		result = append(result, b)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartBeastform(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartCompanionExperience(_ context.Context, e contentstore.DaggerheartCompanionExperienceEntry) error {
	s.companionExperiences[e.ID] = e
	return nil
}

func (s *fakeContentStore) GetDaggerheartCompanionExperience(_ context.Context, id string) (contentstore.DaggerheartCompanionExperienceEntry, error) {
	e, ok := s.companionExperiences[id]
	if !ok {
		return contentstore.DaggerheartCompanionExperienceEntry{}, storage.ErrNotFound
	}
	return e, nil
}

func (s *fakeContentStore) ListDaggerheartCompanionExperiences(_ context.Context) ([]contentstore.DaggerheartCompanionExperienceEntry, error) {
	result := make([]contentstore.DaggerheartCompanionExperienceEntry, 0, len(s.companionExperiences))
	for _, e := range s.companionExperiences {
		result = append(result, e)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartCompanionExperience(_ context.Context, _ string) error {
	return nil
}
