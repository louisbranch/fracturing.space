package game

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

const (
	defaultListSessionsPageSize = pageSmall
	maxListSessionsPageSize     = pageSmall
)

// SessionService implements the game.v1.SessionService gRPC API.
type SessionService struct {
	campaignv1.UnimplementedSessionServiceServer
	app sessionApplication
}

// NewSessionService creates a SessionService with default dependencies.
func NewSessionService(stores Stores) *SessionService {
	return newSessionServiceWithDependencies(stores, time.Now, id.NewID)
}

func newSessionServiceWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
) *SessionService {
	return &SessionService{
		app: newSessionApplicationWithDependencies(stores, clock, idGenerator),
	}
}
