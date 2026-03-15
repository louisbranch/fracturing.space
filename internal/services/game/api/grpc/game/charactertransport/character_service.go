package charactertransport

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
)

const (
	defaultListCharactersPageSize = handler.PageSmall
	maxListCharactersPageSize     = handler.PageSmall
)

// Service implements the game.v1.CharacterService gRPC API.
type Service struct {
	campaignv1.UnimplementedCharacterServiceServer
	app characterApplication
}

// NewService creates a Service with default dependencies.
func NewService(deps Deps) *Service {
	return newServiceWithDependencies(deps, time.Now, id.NewID)
}

func newServiceWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) *Service {
	return &Service{
		app: newCharacterApplicationFromDeps(deps, clock, idGenerator),
	}
}
