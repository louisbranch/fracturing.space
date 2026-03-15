package invitetransport

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
)

// Deps holds all dependencies needed by the invite transport layer.
type Deps struct {
	Auth        authz.PolicyDeps
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Invite      storage.InviteStore
	ClaimIndex  storage.ClaimIndexStore
	Event       storage.EventStore
	Social      socialv1.SocialServiceClient
	Write       domainwriteexec.WritePath
	Applier     projection.Applier
}

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

func newInviteApplication(service *Service) inviteApplication {
	if service == nil {
		return inviteApplication{}
	}
	return service.app
}

func newInviteApplicationWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
	authClient authv1.AuthServiceClient,
	joinGrantVerifier joingrant.Verifier,
) inviteApplication {
	applier := deps.Applier
	if applier.Campaign == nil && deps.Campaign != nil {
		applier = defaultApplier(deps)
	}
	app := inviteApplication{
		auth: deps.Auth,
		stores: inviteApplicationStores{
			Campaign:    deps.Campaign,
			Participant: deps.Participant,
			Character:   deps.Character,
			Invite:      deps.Invite,
			ClaimIndex:  deps.ClaimIndex,
			Event:       deps.Event,
			Social:      deps.Social,
		},
		write:       deps.Write,
		applier:     applier,
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

// defaultApplier builds a minimal projection.Applier from Deps fields so
// callers that do not supply a pre-built Applier get reasonable defaults.
// This mirrors the behavior of the former Stores.Applier() construction.
func defaultApplier(deps Deps) projection.Applier {
	return projection.Applier{
		Campaign:    deps.Campaign,
		Invite:      deps.Invite,
		Participant: deps.Participant,
		Character:   deps.Character,
		ClaimIndex:  deps.ClaimIndex,
	}
}
