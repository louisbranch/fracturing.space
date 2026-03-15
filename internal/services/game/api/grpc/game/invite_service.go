package game

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
)

const (
	defaultListInvitesPageSize = handler.PageSmall
	maxListInvitesPageSize     = handler.PageSmall
)

// InviteService implements the game.v1.InviteService gRPC API.
type InviteService struct {
	campaignv1.UnimplementedInviteServiceServer
	app   inviteApplication
	reads inviteReadDependencies
}

// NewInviteService creates an InviteService. The authClient is optional —
// pass nil when the dependency is not needed (e.g. in tests).
func NewInviteService(stores Stores, authClient authv1.AuthServiceClient) *InviteService {
	return newInviteServiceWithDependencies(stores, time.Now, id.NewID, authClient, nil)
}

func newInviteServiceWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
	authClient authv1.AuthServiceClient,
	joinGrantVerifier joingrant.Verifier,
) *InviteService {
	return &InviteService{
		app:   newInviteApplicationWithDependencies(stores, clock, idGenerator, authClient, joinGrantVerifier),
		reads: newInviteReadDependencies(stores, authClient),
	}
}
