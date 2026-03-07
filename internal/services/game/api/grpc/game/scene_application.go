package game

import "time"

// sceneApplication coordinates scene transport use-cases across focused files
// (lifecycle, character membership, gates, and spotlight operations).
type sceneApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newSceneApplication(service *SceneService) sceneApplication {
	app := sceneApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
