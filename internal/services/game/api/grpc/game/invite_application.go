package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"

	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
)

// inviteApplication coordinates invite transport use-cases across focused
// create, claim, and revoke files.
type inviteApplication struct {
	auth              authz.PolicyDeps
	stores            inviteApplicationStores
	write             domainwriteexec.WritePath
	applier           projection.Applier
	clock             func() time.Time
	idGenerator       func() (string, error)
	authClient        authv1.AuthServiceClient
	joinGrantVerifier joingrant.Verifier
}

type inviteApplicationStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Invite      storage.InviteStore
	ClaimIndex  storage.ClaimIndexStore
	Event       storage.EventStore
	Social      socialv1.SocialServiceClient
}

func newInviteApplication(service *InviteService) inviteApplication {
	if service == nil {
		return inviteApplication{}
	}
	return service.app
}

func newInviteApplicationWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
	authClient authv1.AuthServiceClient,
	joinGrantVerifier joingrant.Verifier,
) inviteApplication {
	app := inviteApplication{
		auth: authz.PolicyDeps{Participant: stores.Participant, Character: stores.Character, Audit: stores.Audit},
		stores: inviteApplicationStores{
			Campaign:    stores.Campaign,
			Participant: stores.Participant,
			Character:   stores.Character,
			Invite:      stores.Invite,
			ClaimIndex:  stores.ClaimIndex,
			Event:       stores.Event,
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
	if joinGrantVerifier != nil {
		app.joinGrantVerifier = joinGrantVerifier
	} else {
		app.joinGrantVerifier = joingrant.EnvVerifier{Now: app.clock}
	}
	return app
}
