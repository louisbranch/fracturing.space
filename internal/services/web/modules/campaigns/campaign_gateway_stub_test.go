package campaigns

import (
	"context"
	"strings"
)

// campaignGatewayStub captures create-campaign input while reusing the shared
// fakeGateway behavior for other gateway methods.
type campaignGatewayStub struct {
	fakeGateway
	createCampaignResult CreateCampaignResult
	createCampaignErr    error
	lastCreateInput      CreateCampaignInput
}

func (g *campaignGatewayStub) CreateCampaign(_ context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	g.lastCreateInput = input
	if g.createCampaignErr != nil {
		return CreateCampaignResult{}, g.createCampaignErr
	}
	created := g.createCampaignResult
	if strings.TrimSpace(created.CampaignID) == "" {
		created.CampaignID = "created"
	}
	return created, nil
}
