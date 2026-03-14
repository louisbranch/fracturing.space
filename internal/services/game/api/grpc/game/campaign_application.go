package game

import (
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// campaignApplication coordinates campaign transport use-cases across focused
// method files (creation, mutation, status transitions, and AI binding).
type campaignApplication struct {
	auth        Stores
	stores      campaignApplicationStores
	write       domainwriteexec.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
	aiClient    aiv1.AgentServiceClient
}

type campaignApplicationStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Session     storage.SessionStore
	Social      socialv1.SocialServiceClient
}

type campaignCommandExecution struct {
	Campaign storage.CampaignStore
	Write    domainwriteexec.WritePath
	Applier  projection.Applier
}

func newCampaignApplication(service *CampaignService) campaignApplication {
	app := campaignApplication{
		auth: service.stores,
		stores: campaignApplicationStores{
			Campaign:    service.stores.Campaign,
			Participant: service.stores.Participant,
			Session:     service.stores.Session,
			Social:      service.stores.Social,
		},
		write:       service.stores.Write,
		applier:     service.stores.Applier(),
		clock:       service.clock,
		idGenerator: service.idGenerator,
		authClient:  service.authClient,
		aiClient:    service.aiClient,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}

func (c campaignApplication) commandExecution() campaignCommandExecution {
	return campaignCommandExecution{
		Campaign: c.stores.Campaign,
		Write:    c.write,
		Applier:  c.applier,
	}
}
