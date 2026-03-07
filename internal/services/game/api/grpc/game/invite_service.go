package game

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
)

const (
	defaultListInvitesPageSize = 10
	maxListInvitesPageSize     = 10
)

// InviteService implements the game.v1.InviteService gRPC API.
type InviteService struct {
	campaignv1.UnimplementedInviteServiceServer
	stores            Stores
	clock             func() time.Time
	idGenerator       func() (string, error)
	authClient        authv1.AuthServiceClient
	joinGrantVerifier joingrant.Verifier
}

// NewInviteService creates an InviteService with default dependencies.
func NewInviteService(stores Stores) *InviteService {
	return &InviteService{
		stores:            stores,
		clock:             time.Now,
		idGenerator:       id.NewID,
		joinGrantVerifier: joingrant.EnvVerifier{Now: time.Now},
	}
}

// NewInviteServiceWithAuth creates an InviteService with an auth client.
func NewInviteServiceWithAuth(stores Stores, authClient authv1.AuthServiceClient) *InviteService {
	service := NewInviteService(stores)
	service.authClient = authClient
	return service
}
