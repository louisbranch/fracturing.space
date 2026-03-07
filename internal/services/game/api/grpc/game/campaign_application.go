package game

import (
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
)

// campaignApplication coordinates campaign transport use-cases across focused
// method files (creation, mutation, status transitions, and AI binding).
type campaignApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	aiClient    aiv1.AgentServiceClient
}

func newCampaignApplication(service *CampaignService) campaignApplication {
	app := campaignApplication{
		stores:      service.stores,
		clock:       service.clock,
		idGenerator: service.idGenerator,
		aiClient:    service.aiClient,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
