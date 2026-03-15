package scenetransport

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// Service implements the game.v1.SceneService gRPC API.
type Service struct {
	campaignv1.UnimplementedSceneServiceServer
	app   sceneApplication
	reads sceneReadDependencies
}

// NewService creates a scene Service with default dependencies.
func NewService(deps Deps) *Service {
	return newServiceWithDependencies(deps, time.Now, id.NewID)
}

func newServiceWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) *Service {
	return &Service{
		app:   newSceneApplicationWithDependencies(deps, clock, idGenerator),
		reads: newSceneReadDependencies(deps),
	}
}
