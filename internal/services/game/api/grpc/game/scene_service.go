package game

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// SceneService implements the game.v1.SceneService gRPC API.
type SceneService struct {
	campaignv1.UnimplementedSceneServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewSceneService creates a SceneService with default dependencies.
func NewSceneService(stores Stores) *SceneService {
	return &SceneService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}
