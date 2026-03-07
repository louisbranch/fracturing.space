package projection

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// fakeWatermarkStore records calls to SaveProjectionWatermark.
type fakeWatermarkStore struct {
	watermarks map[string]storage.ProjectionWatermark
}

func newFakeWatermarkStore() *fakeWatermarkStore {
	return &fakeWatermarkStore{watermarks: make(map[string]storage.ProjectionWatermark)}
}

func (s *fakeWatermarkStore) GetProjectionWatermark(_ context.Context, campaignID string) (storage.ProjectionWatermark, error) {
	wm, ok := s.watermarks[campaignID]
	if !ok {
		return storage.ProjectionWatermark{}, storage.ErrNotFound
	}
	return wm, nil
}

func (s *fakeWatermarkStore) SaveProjectionWatermark(_ context.Context, wm storage.ProjectionWatermark) error {
	s.watermarks[wm.CampaignID] = wm
	return nil
}

func (s *fakeWatermarkStore) ListProjectionWatermarks(_ context.Context) ([]storage.ProjectionWatermark, error) {
	var out []storage.ProjectionWatermark
	for _, wm := range s.watermarks {
		out = append(out, wm)
	}
	return out, nil
}
