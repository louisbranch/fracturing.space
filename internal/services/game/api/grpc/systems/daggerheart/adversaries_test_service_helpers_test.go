package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// newAdversaryTestService returns a root service wired with fake stores for adversary endpoint tests.
func newAdversaryTestService() *DaggerheartService {
	campaignStore := newFakeCampaignStore()
	campaignStore.Campaigns["camp-1"] = storage.CampaignRecord{
		ID:     "camp-1",
		Status: campaign.StatusActive,
		System: bridge.SystemIDDaggerheart,
	}
	campaignStore.Campaigns["camp-non-dh"] = storage.CampaignRecord{
		ID:     "camp-non-dh",
		Status: campaign.StatusActive,
		System: bridge.SystemIDUnspecified,
	}

	sessStore := newFakeSessionStore()
	sessStore.Sessions["camp-1:sess-1"] = storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Status:     session.StatusActive,
	}

	dhStore := newFakeDaggerheartAdversaryStore()
	eventStore := newFakeActionEventStore()

	return &DaggerheartService{
		stores: Stores{
			Campaign:    campaignStore,
			Daggerheart: dhStore,
			Event:       eventStore,
			SessionGate: &fakeSessionGateStore{},
			Session:     sessStore,
			Write:       domainwriteexec.WritePath{Executor: &dynamicDomainEngine{store: eventStore}, Runtime: testRuntime},
		},
		seedFunc: func() (int64, error) { return 42, nil },
	}
}
