package invitetransport

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

// Service implements the game.v1.InviteService gRPC API.
type Service struct {
	campaignv1.UnimplementedInviteServiceServer
	app   inviteApplication
	reads inviteReadDependencies
}

// NewService creates an invite Service with default dependencies.
func NewService(deps Deps, authClient authv1.AuthServiceClient) *Service {
	return newServiceWithDependencies(deps, time.Now, id.NewID, authClient, nil)
}

func newServiceWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
	authClient authv1.AuthServiceClient,
	joinGrantVerifier joingrant.Verifier,
) *Service {
	return &Service{
		app:   newInviteApplicationWithDependencies(deps, clock, idGenerator, authClient, joinGrantVerifier),
		reads: newInviteReadDependencies(deps, authClient),
	}
}
