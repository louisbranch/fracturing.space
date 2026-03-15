package game

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
)

const (
	defaultListParticipantsPageSize = handler.PageSmall
	maxListParticipantsPageSize     = handler.PageSmall
)

// ParticipantService implements the game.v1.ParticipantService gRPC API.
type ParticipantService struct {
	campaignv1.UnimplementedParticipantServiceServer
	app participantApplication
}

// NewParticipantService creates a ParticipantService with default dependencies.
func NewParticipantService(stores Stores, authClients ...authv1.AuthServiceClient) *ParticipantService {
	var authClient authv1.AuthServiceClient
	if len(authClients) > 0 {
		authClient = authClients[0]
	}
	return newParticipantServiceWithDependencies(stores, time.Now, id.NewID, authClient)
}

func newParticipantServiceWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
	authClient authv1.AuthServiceClient,
) *ParticipantService {
	return &ParticipantService{
		app: newParticipantApplicationWithDependencies(stores, clock, idGenerator, authClient),
	}
}
