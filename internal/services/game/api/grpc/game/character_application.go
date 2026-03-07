package game

import "time"

// characterApplication coordinates character transport use-cases across focused
// method files (create, update, delete, control, workflow, and profile patching).
type characterApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newCharacterApplication(service *CharacterService) characterApplication {
	app := characterApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
