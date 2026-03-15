package game

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

const (
	defaultListForksPageSize = pageSmall
	maxListForksPageSize     = pageMedium
	forkEventPageSize        = pageLarge
	forkSnapshotPageSize     = pageLarge
)

// ForkService implements the game.v1.ForkService gRPC API.
type ForkService struct {
	campaignv1.UnimplementedForkServiceServer
	app forkApplication
}

// NewForkService creates a ForkService with default dependencies.
func NewForkService(stores Stores) *ForkService {
	return newForkServiceWithDependencies(stores, time.Now, id.NewID)
}

func newForkServiceWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
) *ForkService {
	return &ForkService{
		app: newForkApplicationWithDependencies(stores, clock, idGenerator),
	}
}
