package campaigns

import (
	"context"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// campaignGatewayStub captures create-campaign input while reusing the shared
// fakeGateway behavior for other gateway methods.
type campaignGatewayStub struct {
	fakeGateway
	createCampaignResult campaignapp.CreateCampaignResult
	createCampaignErr    error
	lastCreateInput      campaignapp.CreateCampaignInput
}

func (g *campaignGatewayStub) CreateCampaign(_ context.Context, input campaignapp.CreateCampaignInput) (campaignapp.CreateCampaignResult, error) {
	g.lastCreateInput = input
	if g.createCampaignErr != nil {
		return campaignapp.CreateCampaignResult{}, g.createCampaignErr
	}
	created := g.createCampaignResult
	if strings.TrimSpace(created.CampaignID) == "" {
		created.CampaignID = "created"
	}
	return created, nil
}
