package participanttransport

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
)

const (
	defaultListParticipantsPageSize = handler.PageSmall
	maxListParticipantsPageSize     = handler.PageSmall
)

// Service implements the game.v1.ParticipantService gRPC API.
type Service struct {
	campaignv1.UnimplementedParticipantServiceServer
	app participantApplication
}

// NewService creates a Service with default dependencies.
func NewService(deps Deps, authClients ...handler.AuthUserClient) *Service {
	var authClient handler.AuthUserClient
	if len(authClients) > 0 {
		authClient = authClients[0]
	}
	return newServiceWithDependencies(deps, time.Now, id.NewID, authClient)
}

func newServiceWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
	authClient handler.AuthUserClient,
) *Service {
	return &Service{
		app: newParticipantApplicationFromDeps(deps, clock, idGenerator, authClient),
	}
}
