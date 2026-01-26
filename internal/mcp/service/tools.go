package service

import (
	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	dualityv1 "github.com/louisbranch/duality-engine/api/gen/go/duality/v1"
	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
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

func registerCampaignTools(mcpServer *mcp.Server, client campaignv1.CampaignServiceClient, getContext func() domain.Context) {
	mcp.AddTool(mcpServer, domain.CampaignCreateTool(), domain.CampaignCreateHandler(client))
	mcp.AddTool(mcpServer, domain.ParticipantCreateTool(), domain.ParticipantCreateHandler(client))
	mcp.AddTool(mcpServer, domain.CharacterCreateTool(), domain.CharacterCreateHandler(client))
	mcp.AddTool(mcpServer, domain.CharacterControlSetTool(), domain.CharacterControlSetHandler(client))
	mcp.AddTool(mcpServer, domain.CharacterSheetGetTool(), domain.CharacterSheetGetHandler(client, getContext))
	mcp.AddTool(mcpServer, domain.CharacterProfilePatchTool(), domain.CharacterProfilePatchHandler(client, getContext))
	mcp.AddTool(mcpServer, domain.CharacterStatePatchTool(), domain.CharacterStatePatchHandler(client, getContext))
}

func registerSessionTools(mcpServer *mcp.Server, client sessionv1.SessionServiceClient) {
	mcp.AddTool(mcpServer, domain.SessionStartTool(), domain.SessionStartHandler(client))
}

// registerContextTools registers context management tools.
func registerContextTools(
	mcpServer *mcp.Server,
	campaignClient campaignv1.CampaignServiceClient,
	sessionClient sessionv1.SessionServiceClient,
	server *Server,
) {
	mcp.AddTool(mcpServer, domain.SetContextTool(), domain.SetContextHandler(
		campaignClient,
		sessionClient,
		server.setContext,
		server.getContext,
	))
}

// registerCampaignResources registers readable campaign MCP resources.
func registerCampaignResources(mcpServer *mcp.Server, client campaignv1.CampaignServiceClient) {
	mcpServer.AddResource(domain.CampaignListResource(), domain.CampaignListResourceHandler(client))
	mcpServer.AddResource(domain.CampaignResource(), domain.CampaignResourceHandler(client))
	mcpServer.AddResource(domain.ParticipantListResource(), domain.ParticipantListResourceHandler(client))
	mcpServer.AddResource(domain.CharacterListResource(), domain.CharacterListResourceHandler(client))
}

// registerSessionResources registers readable session MCP resources.
func registerSessionResources(mcpServer *mcp.Server, client sessionv1.SessionServiceClient) {
	mcpServer.AddResource(domain.SessionListResource(), domain.SessionListResourceHandler(client))
}

// registerContextResources registers readable context MCP resources.
func registerContextResources(mcpServer *mcp.Server, server *Server) {
	mcpServer.AddResource(domain.ContextResource(), domain.ContextResourceHandler(server.getContext))
}
