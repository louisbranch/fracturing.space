package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (s *fakeContentStore) PutDaggerheartClass(_ context.Context, c contentstore.DaggerheartClass) error {
	s.classes[c.ID] = c
	return nil
}

func (s *fakeContentStore) GetDaggerheartClass(_ context.Context, id string) (contentstore.DaggerheartClass, error) {
	c, ok := s.classes[id]
	if !ok {
		return contentstore.DaggerheartClass{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *fakeContentStore) ListDaggerheartClasses(_ context.Context) ([]contentstore.DaggerheartClass, error) {
	result := make([]contentstore.DaggerheartClass, 0, len(s.classes))
	for _, c := range s.classes {
		result = append(result, c)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartClass(_ context.Context, id string) error {
	delete(s.classes, id)
	return nil
}

func (s *fakeContentStore) PutDaggerheartSubclass(_ context.Context, c contentstore.DaggerheartSubclass) error {
	s.subclasses[c.ID] = c
	return nil
}

func (s *fakeContentStore) GetDaggerheartSubclass(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	c, ok := s.subclasses[id]
	if !ok {
		return contentstore.DaggerheartSubclass{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *fakeContentStore) ListDaggerheartSubclasses(_ context.Context) ([]contentstore.DaggerheartSubclass, error) {
	result := make([]contentstore.DaggerheartSubclass, 0, len(s.subclasses))
	for _, c := range s.subclasses {
		result = append(result, c)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartSubclass(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartHeritage(_ context.Context, h contentstore.DaggerheartHeritage) error {
	s.heritages[h.ID] = h
	return nil
}

func (s *fakeContentStore) GetDaggerheartHeritage(_ context.Context, id string) (contentstore.DaggerheartHeritage, error) {
	h, ok := s.heritages[id]
	if !ok {
		return contentstore.DaggerheartHeritage{}, storage.ErrNotFound
	}
	return h, nil
}

func (s *fakeContentStore) ListDaggerheartHeritages(_ context.Context) ([]contentstore.DaggerheartHeritage, error) {
	result := make([]contentstore.DaggerheartHeritage, 0, len(s.heritages))
	for _, h := range s.heritages {
		result = append(result, h)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartHeritage(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartExperience(_ context.Context, e contentstore.DaggerheartExperienceEntry) error {
	s.experiences[e.ID] = e
	return nil
}

func (s *fakeContentStore) GetDaggerheartExperience(_ context.Context, id string) (contentstore.DaggerheartExperienceEntry, error) {
	e, ok := s.experiences[id]
	if !ok {
		return contentstore.DaggerheartExperienceEntry{}, storage.ErrNotFound
	}
	return e, nil
}

func (s *fakeContentStore) ListDaggerheartExperiences(_ context.Context) ([]contentstore.DaggerheartExperienceEntry, error) {
	result := make([]contentstore.DaggerheartExperienceEntry, 0, len(s.experiences))
	for _, e := range s.experiences {
		result = append(result, e)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartExperience(_ context.Context, _ string) error { return nil }
