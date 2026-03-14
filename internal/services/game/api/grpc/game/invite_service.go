package game

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
)

const (
	defaultListInvitesPageSize = pageSmall
	maxListInvitesPageSize     = pageSmall
)

// InviteService implements the game.v1.InviteService gRPC API.
type InviteService struct {
	campaignv1.UnimplementedInviteServiceServer
	stores             Stores
	clock              func() time.Time
	idGenerator        func() (string, error)
	authClient         authv1.AuthServiceClient
	notificationClient notificationsv1.NotificationServiceClient
	joinGrantVerifier  joingrant.Verifier
}

// NewInviteService creates an InviteService. The authClient is optional —
// pass nil when the dependency is not needed (e.g. in tests).
func NewInviteService(stores Stores, authClient authv1.AuthServiceClient, notificationClient ...notificationsv1.NotificationServiceClient) *InviteService {
	service := &InviteService{
		stores:            stores,
		clock:             time.Now,
		idGenerator:       id.NewID,
		joinGrantVerifier: joingrant.EnvVerifier{Now: time.Now},
		authClient:        authClient,
	}
	if len(notificationClient) > 0 {
		service.notificationClient = notificationClient[0]
	}
	return service
}
