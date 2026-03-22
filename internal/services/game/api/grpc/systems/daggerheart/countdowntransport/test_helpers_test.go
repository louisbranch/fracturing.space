package countdowntransport

import (
	"context"
	"sort"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type testCampaignStore struct {
	record storage.CampaignRecord
	err    error
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	if s.err != nil {
		return storage.CampaignRecord{}, s.err
	}
	return s.record, nil
}

type testSessionStore struct {
	record storage.SessionRecord
	err    error
}

func (s testSessionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	if s.err != nil {
		return storage.SessionRecord{}, s.err
	}
	return s.record, nil
}

type testDaggerheartStore struct {
	countdowns map[string]projectionstore.DaggerheartCountdown
	getErr     error
	listErr    error
}

func (s testDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	if s.getErr != nil {
		return projectionstore.DaggerheartCountdown{}, s.getErr
	}
	key := campaignID + ":" + countdownID
	countdown, ok := s.countdowns[key]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (s testDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	result := make([]projectionstore.DaggerheartCountdown, 0, len(s.countdowns))
	for _, countdown := range s.countdowns {
		if countdown.CampaignID == campaignID {
			result = append(result, countdown)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].CountdownID < result[j].CountdownID
	})
	return result, nil
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.Session == nil {
		deps.Session = testSessionStore{record: storage.SessionRecord{
			ID:         "sess-1",
			CampaignID: "camp-1",
			Status:     session.StatusActive,
		}}
	}
	if deps.SessionGate == nil {
		deps.SessionGate = testGateStore{err: storage.ErrNotFound}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = testDaggerheartStore{}
	}
	if deps.NewID == nil {
		deps.NewID = func() (string, error) { return "generated-id", nil }
	}
	return NewHandler(deps)
}

func testContext() context.Context {
	ctx := grpcmeta.WithRequestID(context.Background(), "req-1")
	return grpcmeta.WithInvocationID(ctx, "inv-1")
}
