package game

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// sessionApplication coordinates session transport use-cases across focused
// lifecycle, gate, spotlight, and read files using a narrow dependency bundle
// plus explicit session-owned command executors for write paths.
type sessionApplication struct {
	auth         policyDependencies
	stores       sessionApplicationStores
	commands     sessionCommandExecutor
	gateCommands sessionGateCommandExecutor
	clock        func() time.Time
	idGenerator  func() (string, error)
}

type sessionApplicationStores struct {
	Campaign         storage.CampaignStore
	Participant      storage.ParticipantStore
	Session          storage.SessionStore
	SessionGate      storage.SessionGateStore
	SessionSpotlight storage.SessionSpotlightStore
}

func newSessionApplication(service *SessionService) sessionApplication {
	return newSessionApplicationWithDependencies(service.stores, service.clock, service.idGenerator)
}

func newSessionApplicationWithDependencies(stores Stores, clock func() time.Time, idGenerator func() (string, error)) sessionApplication {
	app := sessionApplication{
		auth: newPolicyDependencies(stores),
		stores: sessionApplicationStores{
			Campaign:         stores.Campaign,
			Participant:      stores.Participant,
			Session:          stores.Session,
			SessionGate:      stores.SessionGate,
			SessionSpotlight: stores.SessionSpotlight,
		},
		commands:     newSessionCommandExecutor(stores.Write, stores.Applier()),
		gateCommands: newSessionGateCommandExecutor(stores.Write, stores.Applier()),
		clock:        clock,
		idGenerator:  idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
