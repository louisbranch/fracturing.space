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
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewCharacterService creates a CharacterService with default dependencies.
func NewCharacterService(stores Stores) *CharacterService {
	return &CharacterService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}
