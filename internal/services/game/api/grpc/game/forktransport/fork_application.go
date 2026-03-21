package forktransport

import (
	"time"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Deps holds all dependencies needed by the fork transport layer.
type Deps struct {
	Auth         authz.PolicyDeps
	Campaign     storage.CampaignStore
	Participant  storage.ParticipantStore
	Character    storage.CharacterStore
	Session      storage.SessionStore
	CampaignFork storage.CampaignForkStore
	Event        storage.EventStore
	Social       socialv1.SocialServiceClient
	Write        domainwrite.WritePath
	Applier      projection.Applier
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
	Social       socialv1.SocialServiceClient
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
		eventReplay: forkEventReplay{
			events:  deps.Event,
			applier: deps.Applier,
			runtime: deps.Write.Runtime,
		},
		write:       deps.Write,
		applier:     deps.Applier,
		clock:       clock,
		idGenerator: idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
