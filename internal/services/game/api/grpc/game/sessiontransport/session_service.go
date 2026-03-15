package sessiontransport

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
)

const (
	defaultListSessionsPageSize = handler.PageSmall
	maxListSessionsPageSize     = handler.PageSmall
)

// SessionService implements the game.v1.SessionService gRPC API.
type SessionService struct {
	campaignv1.UnimplementedSessionServiceServer
	app sessionApplication
}

// NewSessionService creates a SessionService with default dependencies.
func NewSessionService(deps Deps) *SessionService {
	return newSessionServiceWithDependencies(deps, time.Now, id.NewID)
}

func newSessionServiceWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) *SessionService {
	return &SessionService{
		app: newSessionApplicationFromDeps(deps, clock, idGenerator),
	}
}
