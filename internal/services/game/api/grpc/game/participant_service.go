package game

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

const (
	defaultListParticipantsPageSize = 10
	maxListParticipantsPageSize     = 10
)

// ParticipantService implements the game.v1.ParticipantService gRPC API.
type ParticipantService struct {
	campaignv1.UnimplementedParticipantServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewParticipantService creates a ParticipantService with default dependencies.
func NewParticipantService(stores Stores) *ParticipantService {
	return &ParticipantService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}
