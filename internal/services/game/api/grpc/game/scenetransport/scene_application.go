package scenetransport

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Deps holds all dependencies needed by the scene transport layer.
type Deps struct {
	Auth           authz.PolicyDeps
	Campaign       storage.CampaignStore
	Scene          storage.SceneStore
	SceneCharacter storage.SceneCharacterStore
	Write          domainwrite.WritePath
	Applier        projection.Applier
}

// sceneApplication coordinates scene transport use-cases across focused files
// (lifecycle, character membership, gates, and spotlight operations) while
// keeping scene-owned reads and write execution explicit.
type sceneApplication struct {
	auth        authz.PolicyDeps
	stores      sceneApplicationStores
	write       domainwrite.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
}

type sceneApplicationStores struct {
	Campaign storage.CampaignStore
	Scene    storage.SceneStore
}

func newSceneApplication(service *Service) sceneApplication {
	if service == nil {
		return sceneApplication{}
	}
	return service.app
}

func newSceneApplicationWithDependencies(deps Deps, clock func() time.Time, idGenerator func() (string, error)) sceneApplication {
	app := sceneApplication{
		auth: deps.Auth,
		stores: sceneApplicationStores{
			Campaign: deps.Campaign,
			Scene:    deps.Scene,
		},
		write:       deps.Write,
		applier:     deps.Applier,
		clock:       clock,
		idGenerator: idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
