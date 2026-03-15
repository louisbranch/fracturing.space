package game

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// participantApplication coordinates participant transport use-cases across
// focused method files (create, update, delete, and policy helpers).
type participantApplication struct {
	auth        policyDependencies
	stores      participantApplicationStores
	write       domainwriteexec.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
}

type participantApplicationStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Social      socialv1.SocialServiceClient
}

func newParticipantApplicationWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
	authClient authv1.AuthServiceClient,
) participantApplication {
	app := participantApplication{
		auth: newPolicyDependencies(stores),
		stores: participantApplicationStores{
			Campaign:    stores.Campaign,
			Participant: stores.Participant,
			Character:   stores.Character,
			Social:      stores.Social,
		},
		write:       stores.Write,
		applier:     stores.Applier(),
		clock:       clock,
		idGenerator: idGenerator,
		authClient:  authClient,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
