package sessiontransport

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Deps holds the explicit dependencies for the session and communication
// transport subpackage. Session-only callers may leave Scene nil.
type Deps struct {
	Auth               authz.PolicyDeps
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	Character          storage.CharacterStore
	Session            storage.SessionStore
	SessionGate        storage.SessionGateStore
	SessionSpotlight   storage.SessionSpotlightStore
	SessionInteraction storage.SessionInteractionStore
	Scene              storage.SceneStore
	Write              domainwrite.WritePath
	Applier            projection.Applier
}

// sessionApplication coordinates session transport use-cases across focused
// lifecycle, gate, spotlight, and read files using a narrow dependency bundle
// plus explicit session-owned command executors for write paths.
type sessionApplication struct {
	auth         authz.PolicyDeps
	stores       sessionApplicationStores
	commands     sessionCommandExecutor
	gateCommands sessionGateCommandExecutor
	clock        func() time.Time
	idGenerator  func() (string, error)
}

type sessionApplicationStores struct {
	Campaign         storage.CampaignStore
	Participant      storage.ParticipantStore
	Character        storage.CharacterStore
	Session          storage.SessionStore
	SessionGate      storage.SessionGateStore
	SessionSpotlight storage.SessionSpotlightStore
}

func newSessionApplication(service *SessionService) sessionApplication {
	if service == nil {
		return sessionApplication{}
	}
	return service.app
}

func newSessionApplicationFromDeps(deps Deps, clock func() time.Time, idGenerator func() (string, error)) sessionApplication {
	auth := deps.Auth
	if auth.Participant == nil {
		auth = authz.PolicyDeps{Participant: deps.Participant, Character: deps.Character, Audit: auth.Audit}
	}
	app := sessionApplication{
		auth: auth,
		stores: sessionApplicationStores{
			Campaign:         deps.Campaign,
			Participant:      deps.Participant,
			Character:        deps.Character,
			Session:          deps.Session,
			SessionGate:      deps.SessionGate,
			SessionSpotlight: deps.SessionSpotlight,
		},
		commands:     newSessionCommandExecutor(deps.Write, deps.Applier),
		gateCommands: newSessionGateCommandExecutor(deps.Write, deps.Applier),
		clock:        clock,
		idGenerator:  idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
