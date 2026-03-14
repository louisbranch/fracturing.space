package game

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// sceneApplication coordinates scene transport use-cases across focused files
// (lifecycle, character membership, gates, and spotlight operations) while
// keeping scene-owned reads and write execution explicit.
type sceneApplication struct {
	auth        Stores
	stores      sceneApplicationStores
	write       domainwriteexec.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
}

type sceneApplicationStores struct {
	Campaign storage.CampaignStore
	Scene    storage.SceneStore
}

func newSceneApplication(service *SceneService) sceneApplication {
	app := sceneApplication{
		auth: service.stores,
		stores: sceneApplicationStores{
			Campaign: service.stores.Campaign,
			Scene:    service.stores.Scene,
		},
		write:       service.stores.Write,
		applier:     service.stores.Applier(),
		clock:       service.clock,
		idGenerator: service.idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
