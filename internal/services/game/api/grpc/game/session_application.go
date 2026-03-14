package game

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// sessionApplication coordinates session transport use-cases across focused
// lifecycle, gate, and spotlight files using a narrow dependency bundle for
// session-owned reads plus explicit write/applier seams.
type sessionApplication struct {
	auth        Stores
	stores      sessionApplicationStores
	write       domainwriteexec.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
}

type sessionApplicationStores struct {
	Campaign         storage.CampaignStore
	Session          storage.SessionStore
	SessionGate      storage.SessionGateStore
	SessionSpotlight storage.SessionSpotlightStore
}

func newSessionApplication(service *SessionService) sessionApplication {
	app := sessionApplication{
		auth: service.stores,
		stores: sessionApplicationStores{
			Campaign:         service.stores.Campaign,
			Session:          service.stores.Session,
			SessionGate:      service.stores.SessionGate,
			SessionSpotlight: service.stores.SessionSpotlight,
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
