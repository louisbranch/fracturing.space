package game

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

const (
	defaultListForksPageSize = 10
	maxListForksPageSize     = 50
	forkEventPageSize        = 200
	forkSnapshotPageSize     = 200
)

// ForkService implements the game.v1.ForkService gRPC API.
type ForkService struct {
	campaignv1.UnimplementedForkServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewForkService creates a ForkService with default dependencies.
func NewForkService(stores Stores) *ForkService {
	return &ForkService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}
