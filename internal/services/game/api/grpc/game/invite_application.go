package game

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
)

// inviteApplication coordinates invite transport use-cases across focused
// create, claim, and revoke files.
type inviteApplication struct {
	stores            Stores
	clock             func() time.Time
	idGenerator       func() (string, error)
	authClient        authv1.AuthServiceClient
	joinGrantVerifier joingrant.Verifier
}

func newInviteApplication(service *InviteService) inviteApplication {
	app := inviteApplication{
		stores:      service.stores,
		clock:       service.clock,
		idGenerator: service.idGenerator,
		authClient:  service.authClient,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	if service.joinGrantVerifier != nil {
		app.joinGrantVerifier = service.joinGrantVerifier
	} else {
		app.joinGrantVerifier = joingrant.EnvVerifier{Now: app.clock}
	}
	return app
}
