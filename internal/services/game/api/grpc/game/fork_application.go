package game

import (
	"time"
)

type forkApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newForkApplication(service *ForkService) forkApplication {
	app := forkApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
