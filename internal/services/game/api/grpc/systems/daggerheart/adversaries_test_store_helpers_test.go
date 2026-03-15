package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// fakeDaggerheartAdversaryStore extends the shared Daggerheart fake store with adversary CRUD state.
type fakeDaggerheartAdversaryStore struct {
	fakeDaggerheartStore
	adversaries map[string]projectionstore.DaggerheartAdversary
}

// newFakeDaggerheartAdversaryStore returns a projection-backed fake store for adversary root endpoint tests.
func newFakeDaggerheartAdversaryStore() *fakeDaggerheartAdversaryStore {
	return &fakeDaggerheartAdversaryStore{
		fakeDaggerheartStore: *newFakeDaggerheartStore(),
		adversaries:          make(map[string]projectionstore.DaggerheartAdversary),
	}
}

func (s *fakeDaggerheartAdversaryStore) PutDaggerheartAdversary(_ context.Context, a projectionstore.DaggerheartAdversary) error {
	s.adversaries[a.CampaignID+":"+a.AdversaryID] = a
	return nil
}

func (s *fakeDaggerheartAdversaryStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	a, ok := s.adversaries[campaignID+":"+adversaryID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return a, nil
}

func (s *fakeDaggerheartAdversaryStore) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
	var result []projectionstore.DaggerheartAdversary
	for _, a := range s.adversaries {
		if a.CampaignID != campaignID {
			continue
		}
		if sessionID != "" && a.SessionID != sessionID {
			continue
		}
		result = append(result, a)
	}
	return result, nil
}

func (s *fakeDaggerheartAdversaryStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	delete(s.adversaries, campaignID+":"+adversaryID)
	return nil
}
