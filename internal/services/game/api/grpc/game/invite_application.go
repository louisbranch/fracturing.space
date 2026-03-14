package game

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
)

// inviteApplication coordinates invite transport use-cases across focused
// create, claim, and revoke files.
type inviteApplication struct {
	auth               Stores
	stores             inviteApplicationStores
	write              domainwriteexec.WritePath
	applier            projection.Applier
	clock              func() time.Time
	idGenerator        func() (string, error)
	authClient         authv1.AuthServiceClient
	notificationClient notificationsv1.NotificationServiceClient
	joinGrantVerifier  joingrant.Verifier
}

type inviteApplicationStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Invite      storage.InviteStore
	ClaimIndex  storage.ClaimIndexStore
	Event       storage.EventStore
	Social      socialv1.SocialServiceClient
}

func newInviteApplication(service *InviteService) inviteApplication {
	app := inviteApplication{
		auth: service.stores,
		stores: inviteApplicationStores{
			Campaign:    service.stores.Campaign,
			Participant: service.stores.Participant,
			Invite:      service.stores.Invite,
			ClaimIndex:  service.stores.ClaimIndex,
			Event:       service.stores.Event,
			Social:      service.stores.Social,
		},
		write:              service.stores.Write,
		applier:            service.stores.Applier(),
		clock:              service.clock,
		idGenerator:        service.idGenerator,
		authClient:         service.authClient,
		notificationClient: service.notificationClient,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	if service.joinGrantVerifier != nil {
		app.joinGrantVerifier = service.joinGrantVerifier
	} else {
		app.joinGrantVerifier = joingrant.EnvVerifier{Now: app.clock}
	}
	return app
}
