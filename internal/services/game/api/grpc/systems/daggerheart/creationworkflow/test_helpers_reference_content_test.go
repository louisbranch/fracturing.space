package creationworkflow

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func (s *testContentStore) GetDaggerheartClass(_ context.Context, id string) (contentstore.DaggerheartClass, error) {
	return mapGet(s.classes, id)
}

func (s *testContentStore) ListDaggerheartClasses(_ context.Context) ([]contentstore.DaggerheartClass, error) {
	return mapList(s.classes)
}

func (s *testContentStore) GetDaggerheartSubclass(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	return mapGet(s.subclasses, id)
}

func (s *testContentStore) ListDaggerheartSubclasses(_ context.Context) ([]contentstore.DaggerheartSubclass, error) {
	return mapList(s.subclasses)
}

func (s *testContentStore) GetDaggerheartHeritage(_ context.Context, id string) (contentstore.DaggerheartHeritage, error) {
	return mapGet(s.heritages, id)
}

func (s *testContentStore) ListDaggerheartHeritages(_ context.Context) ([]contentstore.DaggerheartHeritage, error) {
	return mapList(s.heritages)
}

func (s *testContentStore) GetDaggerheartExperience(_ context.Context, id string) (contentstore.DaggerheartExperienceEntry, error) {
	return mapGet(s.experiences, id)
}

func (s *testContentStore) ListDaggerheartExperiences(_ context.Context) ([]contentstore.DaggerheartExperienceEntry, error) {
	return mapList(s.experiences)
}
