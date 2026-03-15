package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (s *fakeContentStore) PutDaggerheartLootEntry(_ context.Context, l contentstore.DaggerheartLootEntry) error {
	s.lootEntries[l.ID] = l
	return nil
}

func (s *fakeContentStore) GetDaggerheartLootEntry(_ context.Context, id string) (contentstore.DaggerheartLootEntry, error) {
	l, ok := s.lootEntries[id]
	if !ok {
		return contentstore.DaggerheartLootEntry{}, storage.ErrNotFound
	}
	return l, nil
}

func (s *fakeContentStore) ListDaggerheartLootEntries(_ context.Context) ([]contentstore.DaggerheartLootEntry, error) {
	result := make([]contentstore.DaggerheartLootEntry, 0, len(s.lootEntries))
	for _, l := range s.lootEntries {
		result = append(result, l)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartLootEntry(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartWeapon(_ context.Context, w contentstore.DaggerheartWeapon) error {
	s.weapons[w.ID] = w
	return nil
}

func (s *fakeContentStore) GetDaggerheartWeapon(_ context.Context, id string) (contentstore.DaggerheartWeapon, error) {
	w, ok := s.weapons[id]
	if !ok {
		return contentstore.DaggerheartWeapon{}, storage.ErrNotFound
	}
	return w, nil
}

func (s *fakeContentStore) ListDaggerheartWeapons(_ context.Context) ([]contentstore.DaggerheartWeapon, error) {
	result := make([]contentstore.DaggerheartWeapon, 0, len(s.weapons))
	for _, w := range s.weapons {
		result = append(result, w)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartWeapon(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartArmor(_ context.Context, a contentstore.DaggerheartArmor) error {
	s.armor[a.ID] = a
	return nil
}

func (s *fakeContentStore) GetDaggerheartArmor(_ context.Context, id string) (contentstore.DaggerheartArmor, error) {
	a, ok := s.armor[id]
	if !ok {
		return contentstore.DaggerheartArmor{}, storage.ErrNotFound
	}
	return a, nil
}

func (s *fakeContentStore) ListDaggerheartArmor(_ context.Context) ([]contentstore.DaggerheartArmor, error) {
	result := make([]contentstore.DaggerheartArmor, 0, len(s.armor))
	for _, a := range s.armor {
		result = append(result, a)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartArmor(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartItem(_ context.Context, i contentstore.DaggerheartItem) error {
	s.items[i.ID] = i
	return nil
}

func (s *fakeContentStore) GetDaggerheartItem(_ context.Context, id string) (contentstore.DaggerheartItem, error) {
	i, ok := s.items[id]
	if !ok {
		return contentstore.DaggerheartItem{}, storage.ErrNotFound
	}
	return i, nil
}

func (s *fakeContentStore) ListDaggerheartItems(_ context.Context) ([]contentstore.DaggerheartItem, error) {
	result := make([]contentstore.DaggerheartItem, 0, len(s.items))
	for _, i := range s.items {
		result = append(result, i)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartItem(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartEnvironment(_ context.Context, e contentstore.DaggerheartEnvironment) error {
	s.environments[e.ID] = e
	return nil
}

func (s *fakeContentStore) GetDaggerheartEnvironment(_ context.Context, id string) (contentstore.DaggerheartEnvironment, error) {
	e, ok := s.environments[id]
	if !ok {
		return contentstore.DaggerheartEnvironment{}, storage.ErrNotFound
	}
	return e, nil
}

func (s *fakeContentStore) ListDaggerheartEnvironments(_ context.Context) ([]contentstore.DaggerheartEnvironment, error) {
	result := make([]contentstore.DaggerheartEnvironment, 0, len(s.environments))
	for _, e := range s.environments {
		result = append(result, e)
	}
	return result, nil
}

func (s *fakeContentStore) DeleteDaggerheartEnvironment(_ context.Context, _ string) error {
	return nil
}
