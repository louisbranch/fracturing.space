package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"

	"time"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type forkApplication struct {
	auth        authz.PolicyDeps
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
	Character    storage.CharacterStore
	Session      storage.SessionStore
	CampaignFork storage.CampaignForkStore
	Event        storage.EventStore
	Social       socialv1.SocialServiceClient
}

func newForkApplicationWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
) forkApplication {
	app := forkApplication{
		auth: authz.PolicyDeps{Participant: stores.Participant, Character: stores.Character, Audit: stores.Audit},
		stores: forkApplicationStores{
			Campaign:     stores.Campaign,
			Participant:  stores.Participant,
			Character:    stores.Character,
			Session:      stores.Session,
			CampaignFork: stores.CampaignFork,
			Event:        stores.Event,
			Social:       stores.Social,
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
