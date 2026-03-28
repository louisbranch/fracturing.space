package forktransport

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler/social"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/journalimport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Deps holds all dependencies needed by the fork transport layer.
type Deps struct {
	Auth          authz.PolicyDeps
	Campaign      storage.CampaignStore
	Participant   storage.ParticipantStore
	Character     storage.CharacterStore
	Session       storage.SessionStore
	CampaignFork  storage.CampaignForkStore
	Event         storage.EventStore
	Social        social.ProfileClient
	Write         domainwrite.WritePath
	Applier       projection.Applier
	EventRegistry *event.Registry
	Importer      journalimport.Importer
}

type forkApplication struct {
	auth        authz.PolicyDeps
	stores      forkApplicationStores
	eventReplay forkEventReplay
	write       domainwrite.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
}

type forkApplicationStores struct {
	Campaign     storage.CampaignStore
	Participant  storage.ParticipantStore
	Character    storage.CharacterStore
	Session      storage.SessionStore
	CampaignFork storage.CampaignForkStore
	Event        storage.EventStore
	Social       social.ProfileClient
}

func newForkApplicationWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) forkApplication {
	app := forkApplication{
		auth: deps.Auth,
		stores: forkApplicationStores{
			Campaign:     deps.Campaign,
			Participant:  deps.Participant,
			Character:    deps.Character,
			Session:      deps.Session,
			CampaignFork: deps.CampaignFork,
			Event:        deps.Event,
			Social:       deps.Social,
		},
		write:       deps.Write,
		applier:     deps.Applier,
		clock:       clock,
		idGenerator: idGenerator,
	}
	importer := deps.Importer
	if importer == nil {
		defaultImporter := journalimport.NewService(deps.Event, deps.Applier, deps.Write.Runtime, deps.EventRegistry)
		importer = defaultImporter
	}
	app.eventReplay = forkEventReplay{
		events:   deps.Event,
		importer: importer,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
