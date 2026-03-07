package game

import "time"

// participantApplication coordinates participant transport use-cases across
// focused method files (create, update, delete, and policy helpers).
type participantApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newParticipantApplication(service *ParticipantService) participantApplication {
	app := participantApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
