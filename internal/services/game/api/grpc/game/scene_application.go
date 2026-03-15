package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"

	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// sceneApplication coordinates scene transport use-cases across focused files
// (lifecycle, character membership, gates, and spotlight operations) while
// keeping scene-owned reads and write execution explicit.
type sceneApplication struct {
	auth        authz.PolicyDeps
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
	if service == nil {
		return sceneApplication{}
	}
	return service.app
}

func newSceneApplicationWithDependencies(stores Stores, clock func() time.Time, idGenerator func() (string, error)) sceneApplication {
	app := sceneApplication{
		auth: authz.PolicyDeps{Participant: stores.Participant, Character: stores.Character, Audit: stores.Audit},
		stores: sceneApplicationStores{
			Campaign: stores.Campaign,
			Scene:    stores.Scene,
		},
		write:       stores.Write,
		applier:     stores.Applier(),
		clock:       clock,
		idGenerator: idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
