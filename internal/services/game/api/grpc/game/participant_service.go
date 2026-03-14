package game

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

const (
	defaultListParticipantsPageSize = pageSmall
	maxListParticipantsPageSize     = pageSmall
)

// ParticipantService implements the game.v1.ParticipantService gRPC API.
type ParticipantService struct {
	campaignv1.UnimplementedParticipantServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
}

// NewParticipantService creates a ParticipantService with default dependencies.
func NewParticipantService(stores Stores, authClients ...authv1.AuthServiceClient) *ParticipantService {
	var authClient authv1.AuthServiceClient
	if len(authClients) > 0 {
		authClient = authClients[0]
	}
	return &ParticipantService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
		authClient:  authClient,
	}
}
