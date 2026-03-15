package game

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type forkApplication struct {
	auth        policyDependencies
	stores      forkApplicationStores
	eventReplay forkEventReplay
	write       domainwriteexec.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
}

type forkApplicationStores struct {
	Campaign     storage.CampaignStore
	Participant  storage.ParticipantStore
	Session      storage.SessionStore
	CampaignFork storage.CampaignForkStore
	Event        storage.EventStore
}

func newForkApplicationWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
) forkApplication {
	app := forkApplication{
		auth: newPolicyDependencies(stores),
		stores: forkApplicationStores{
			Campaign:     stores.Campaign,
			Participant:  stores.Participant,
			Session:      stores.Session,
			CampaignFork: stores.CampaignFork,
			Event:        stores.Event,
		},
		eventReplay: forkEventReplay{
			events:  stores.Event,
			applier: stores.Applier(),
			runtime: stores.Write.Runtime,
		},
		write:       stores.Write,
		applier:     stores.Applier(),
		clock:       clock,
		idGenerator: idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
