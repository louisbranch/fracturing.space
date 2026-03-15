package game

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

const (
	defaultListCharactersPageSize = pageSmall
	maxListCharactersPageSize     = pageSmall
)

// CharacterService implements the game.v1.CharacterService gRPC API.
type CharacterService struct {
	campaignv1.UnimplementedCharacterServiceServer
	app characterApplication
}

// NewCharacterService creates a CharacterService with default dependencies.
func NewCharacterService(stores Stores) *CharacterService {
	return newCharacterServiceWithDependencies(stores, time.Now, id.NewID)
}

func newCharacterServiceWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
) *CharacterService {
	return &CharacterService{
		app: newCharacterApplicationWithDependencies(stores, clock, idGenerator),
	}
}
