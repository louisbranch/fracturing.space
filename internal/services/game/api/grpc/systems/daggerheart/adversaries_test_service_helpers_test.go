package daggerheart

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

const (
	testAdversaryEntryGoblinID = "entry-goblin"
	testAdversaryEntryOrcID    = "entry-orc"
	testAdversarySessionID     = "sess-1"
	testAdversaryAltSessionID  = "sess-2"
	testAdversarySceneID       = "scene-1"
	testAdversaryAltSceneID    = "scene-2"
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
		ID:         testAdversarySessionID,
		CampaignID: "camp-1",
		Status:     session.StatusActive,
	}
	sessStore.Sessions["camp-1:sess-2"] = storage.SessionRecord{
		ID:         testAdversaryAltSessionID,
		CampaignID: "camp-1",
		Status:     session.StatusActive,
	}

	dhStore := newFakeDaggerheartAdversaryStore()
	eventStore := newFakeActionEventStore()
	contentStore := newFakeContentStore()
	contentStore.adversaryEntries[testAdversaryEntryGoblinID] = contentstore.DaggerheartAdversaryEntry{
		ID:              testAdversaryEntryGoblinID,
		Name:            "Goblin",
		Role:            "bruiser",
		Difficulty:      11,
		MajorThreshold:  6,
		SevereThreshold: 12,
		HP:              6,
		Stress:          2,
		Armor:           1,
	}
	contentStore.adversaryEntries[testAdversaryEntryOrcID] = contentstore.DaggerheartAdversaryEntry{
		ID:              testAdversaryEntryOrcID,
		Name:            "Orc",
		Role:            "soldier",
		Difficulty:      12,
		MajorThreshold:  7,
		SevereThreshold: 14,
		HP:              8,
		Stress:          3,
		Armor:           1,
	}

	return &DaggerheartService{
		stores: Stores{
			Campaign:    campaignStore,
			Content:     contentStore,
			Daggerheart: dhStore,
			Event:       eventStore,
			SessionGate: &fakeSessionGateStore{},
			Session:     sessStore,
			Write:       domainwriteexec.WritePath{Executor: &dynamicDomainEngine{store: eventStore}, Runtime: testRuntime},
		},
		seedFunc: func() (int64, error) { return 42, nil },
	}
}

func adversaryCreateRequest(entryID string) *pb.DaggerheartCreateAdversaryRequest {
	return &pb.DaggerheartCreateAdversaryRequest{
		CampaignId:       "camp-1",
		SessionId:        testAdversarySessionID,
		SceneId:          testAdversarySceneID,
		AdversaryEntryId: entryID,
	}
}
