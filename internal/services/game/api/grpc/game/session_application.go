package game

import "time"

// sessionApplication coordinates session transport use-cases across focused
// lifecycle, gate, and spotlight files.
type sessionApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newSessionApplication(service *SessionService) sessionApplication {
	app := sessionApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
