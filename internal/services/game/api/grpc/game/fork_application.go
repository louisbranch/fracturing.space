package game

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type forkApplication struct {
	auth        Stores
	stores      forkApplicationStores
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

func newForkApplication(service *ForkService) forkApplication {
	app := forkApplication{
		auth: service.stores,
		stores: forkApplicationStores{
			Campaign:     service.stores.Campaign,
			Participant:  service.stores.Participant,
			Session:      service.stores.Session,
			CampaignFork: service.stores.CampaignFork,
			Event:        service.stores.Event,
		},
		write:       service.stores.Write,
		applier:     service.stores.Applier(),
		clock:       service.clock,
		idGenerator: service.idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
