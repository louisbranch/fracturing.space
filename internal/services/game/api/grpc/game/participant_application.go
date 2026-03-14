package game

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
)

// participantApplication coordinates participant transport use-cases across
// focused method files (create, update, delete, and policy helpers).
type participantApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
}

func newParticipantApplication(service *ParticipantService) participantApplication {
	app := participantApplication{
		stores:      service.stores,
		clock:       service.clock,
		idGenerator: service.idGenerator,
		authClient:  service.authClient,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
