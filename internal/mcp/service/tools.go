package service

import (
	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	dualityv1 "github.com/louisbranch/duality-engine/api/gen/go/duality/v1"
	"github.com/louisbranch/duality-engine/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerDualityTools(mcpServer *mcp.Server, client dualityv1.DualityServiceClient) {
	mcp.AddTool(mcpServer, domain.ActionRollTool(), domain.ActionRollHandler(client))
	mcp.AddTool(mcpServer, domain.DualityOutcomeTool(), domain.DualityOutcomeHandler(client))
	mcp.AddTool(mcpServer, domain.DualityExplainTool(), domain.DualityExplainHandler(client))
	mcp.AddTool(mcpServer, domain.DualityProbabilityTool(), domain.DualityProbabilityHandler(client))
	mcp.AddTool(mcpServer, domain.RulesVersionTool(), domain.RulesVersionHandler(client))
	mcp.AddTool(mcpServer, domain.RollDiceTool(), domain.RollDiceHandler(client))
}

func registerCampaignTools(mcpServer *mcp.Server, client campaignv1.CampaignServiceClient) {
	mcp.AddTool(mcpServer, domain.CampaignCreateTool(), domain.CampaignCreateHandler(client))
	mcp.AddTool(mcpServer, domain.ParticipantCreateTool(), domain.ParticipantCreateHandler(client))
	mcp.AddTool(mcpServer, domain.ActorCreateTool(), domain.ActorCreateHandler(client))
}

// registerCampaignResources registers readable campaign MCP resources.
func registerCampaignResources(mcpServer *mcp.Server, client campaignv1.CampaignServiceClient) {
	mcpServer.AddResource(domain.CampaignListResource(), domain.CampaignListResourceHandler(client))
	mcpServer.AddResource(domain.ParticipantListResource(), domain.ParticipantListResourceHandler(client))
	mcpServer.AddResource(domain.ActorListResource(), domain.ActorListResourceHandler(client))
}
