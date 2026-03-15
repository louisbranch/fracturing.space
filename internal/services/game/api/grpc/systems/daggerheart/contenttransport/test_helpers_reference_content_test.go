package contenttransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func (s *fakeContentStore) GetDaggerheartClass(_ context.Context, id string) (contentstore.DaggerheartClass, error) {
	return mapGet(s.classes, id)
}

func (s *fakeContentStore) ListDaggerheartClasses(_ context.Context) ([]contentstore.DaggerheartClass, error) {
	return mapList(s.classes)
}

func (s *fakeContentStore) GetDaggerheartSubclass(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	return mapGet(s.subclasses, id)
}

func (s *fakeContentStore) ListDaggerheartSubclasses(_ context.Context) ([]contentstore.DaggerheartSubclass, error) {
	return mapList(s.subclasses)
}

func (s *fakeContentStore) GetDaggerheartHeritage(_ context.Context, id string) (contentstore.DaggerheartHeritage, error) {
	return mapGet(s.heritages, id)
}

func (s *fakeContentStore) ListDaggerheartHeritages(_ context.Context) ([]contentstore.DaggerheartHeritage, error) {
	return mapList(s.heritages)
}

func (s *fakeContentStore) GetDaggerheartExperience(_ context.Context, id string) (contentstore.DaggerheartExperienceEntry, error) {
	return mapGet(s.experiences, id)
}

func (s *fakeContentStore) ListDaggerheartExperiences(_ context.Context) ([]contentstore.DaggerheartExperienceEntry, error) {
	return mapList(s.experiences)
}
