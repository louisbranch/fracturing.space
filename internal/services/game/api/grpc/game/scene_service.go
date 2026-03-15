package game

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// SceneService implements the game.v1.SceneService gRPC API.
type SceneService struct {
	campaignv1.UnimplementedSceneServiceServer
	app   sceneApplication
	reads sceneReadDependencies
}

// NewSceneService creates a SceneService with default dependencies.
func NewSceneService(stores Stores) *SceneService {
	return newSceneServiceWithDependencies(stores, time.Now, id.NewID)
}

func newSceneServiceWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
) *SceneService {
	return &SceneService{
		app:   newSceneApplicationWithDependencies(stores, clock, idGenerator),
		reads: newSceneReadDependencies(stores),
	}
}
