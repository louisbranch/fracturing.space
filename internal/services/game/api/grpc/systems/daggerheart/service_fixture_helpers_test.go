package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// newTestService creates a handler with a fixed seed generator.
func newTestService(seed int64) *DaggerheartService {
	return &DaggerheartService{
		seedFunc: func() (int64, error) {
			return seed, nil
		},
	}
}

// intPointer converts a difficulty pointer to the duality package type.
func intPointer(value *int32) *int {
	if value == nil {
		return nil
	}

	converted := int(*value)
	return &converted
}

func stringPointer(value string) *string {
	return &value
}

func validDaggerheartStoresForConstructorTests() Stores {
	return Stores{
		Campaign:         &fakeCampaignStore{},
		Character:        &fakeCharacterStore{},
		Content:          newFakeContentStore(),
		Session:          &fakeSessionStore{},
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      &fakeDaggerheartStore{},
		Event:            &fakeEventStore{},
		Events:           event.NewRegistry(),
		Write:            domainwrite.WritePath{Executor: &fakeDomainEngine{}, Runtime: testRuntime},
	}
}
