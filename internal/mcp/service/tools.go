package service

import (
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerDaggerheartTools(mcpServer *mcp.Server, client daggerheartv1.DaggerheartServiceClient) {
	mcp.AddTool(mcpServer, domain.ActionRollTool(), domain.ActionRollHandler(client))
	mcp.AddTool(mcpServer, domain.DualityOutcomeTool(), domain.DualityOutcomeHandler(client))
	mcp.AddTool(mcpServer, domain.DualityExplainTool(), domain.DualityExplainHandler(client))
	mcp.AddTool(mcpServer, domain.DualityProbabilityTool(), domain.DualityProbabilityHandler(client))
	mcp.AddTool(mcpServer, domain.RulesVersionTool(), domain.RulesVersionHandler(client))
	mcp.AddTool(mcpServer, domain.RollDiceTool(), domain.RollDiceHandler(client))
}

func registerCampaignTools(
	mcpServer *mcp.Server,
	campaignClient statev1.CampaignServiceClient,
	participantClient statev1.ParticipantServiceClient,
	characterClient statev1.CharacterServiceClient,
	snapshotClient statev1.SnapshotServiceClient,
	getContext func() domain.Context,
	notify domain.ResourceUpdateNotifier,
) {
	mcp.AddTool(mcpServer, domain.CampaignCreateTool(), domain.CampaignCreateHandler(campaignClient, notify))
	mcp.AddTool(mcpServer, domain.CampaignEndTool(), domain.CampaignEndHandler(campaignClient, getContext, notify))
	mcp.AddTool(mcpServer, domain.CampaignArchiveTool(), domain.CampaignArchiveHandler(campaignClient, getContext, notify))
	mcp.AddTool(mcpServer, domain.CampaignRestoreTool(), domain.CampaignRestoreHandler(campaignClient, getContext, notify))
	mcp.AddTool(mcpServer, domain.ParticipantCreateTool(), domain.ParticipantCreateHandler(participantClient, getContext, notify))
	mcp.AddTool(mcpServer, domain.ParticipantUpdateTool(), domain.ParticipantUpdateHandler(participantClient, getContext, notify))
	mcp.AddTool(mcpServer, domain.ParticipantDeleteTool(), domain.ParticipantDeleteHandler(participantClient, getContext, notify))
	mcp.AddTool(mcpServer, domain.CharacterCreateTool(), domain.CharacterCreateHandler(characterClient, notify))
	mcp.AddTool(mcpServer, domain.CharacterUpdateTool(), domain.CharacterUpdateHandler(characterClient, notify))
	mcp.AddTool(mcpServer, domain.CharacterDeleteTool(), domain.CharacterDeleteHandler(characterClient, notify))
	mcp.AddTool(mcpServer, domain.CharacterControlSetTool(), domain.CharacterControlSetHandler(characterClient, notify))
	mcp.AddTool(mcpServer, domain.CharacterSheetGetTool(), domain.CharacterSheetGetHandler(characterClient, getContext))
	mcp.AddTool(mcpServer, domain.CharacterProfilePatchTool(), domain.CharacterProfilePatchHandler(characterClient, getContext, notify))
	mcp.AddTool(mcpServer, domain.CharacterStatePatchTool(), domain.CharacterStatePatchHandler(snapshotClient, getContext, notify))
}

func registerSessionTools(mcpServer *mcp.Server, client statev1.SessionServiceClient, getContext func() domain.Context, notify domain.ResourceUpdateNotifier) {
	mcp.AddTool(mcpServer, domain.SessionStartTool(), domain.SessionStartHandler(client, notify))
	mcp.AddTool(mcpServer, domain.SessionEndTool(), domain.SessionEndHandler(client, getContext, notify))
}

func registerEventTools(mcpServer *mcp.Server, client statev1.EventServiceClient, getContext func() domain.Context) {
	mcp.AddTool(mcpServer, domain.EventListTool(), domain.EventListHandler(client, getContext))
}

func registerForkTools(mcpServer *mcp.Server, client statev1.ForkServiceClient, notify domain.ResourceUpdateNotifier) {
	mcp.AddTool(mcpServer, domain.CampaignForkTool(), domain.CampaignForkHandler(client, notify))
	mcp.AddTool(mcpServer, domain.CampaignLineageTool(), domain.CampaignLineageHandler(client))
}

// registerContextTools registers context management tools.
func registerContextTools(
	mcpServer *mcp.Server,
	campaignClient statev1.CampaignServiceClient,
	sessionClient statev1.SessionServiceClient,
	participantClient statev1.ParticipantServiceClient,
	server *Server,
	notify domain.ResourceUpdateNotifier,
) {
	mcp.AddTool(mcpServer, domain.SetContextTool(), domain.SetContextHandler(
		campaignClient,
		sessionClient,
		participantClient,
		server.setContext,
		server.getContext,
		notify,
	))
}

// registerCampaignResources registers readable campaign MCP resources.
func registerCampaignResources(
	mcpServer *mcp.Server,
	campaignClient statev1.CampaignServiceClient,
	participantClient statev1.ParticipantServiceClient,
	characterClient statev1.CharacterServiceClient,
) {
	mcpServer.AddResource(domain.CampaignListResource(), domain.CampaignListResourceHandler(campaignClient))
	mcpServer.AddResourceTemplate(domain.CampaignResourceTemplate(), domain.CampaignResourceHandler(campaignClient))
	mcpServer.AddResourceTemplate(domain.ParticipantListResourceTemplate(), domain.ParticipantListResourceHandler(participantClient))
	mcpServer.AddResourceTemplate(domain.CharacterListResourceTemplate(), domain.CharacterListResourceHandler(characterClient))
}

// registerSessionResources registers readable session MCP resources.
func registerSessionResources(mcpServer *mcp.Server, client statev1.SessionServiceClient) {
	mcpServer.AddResourceTemplate(domain.SessionListResourceTemplate(), domain.SessionListResourceHandler(client))
}

// registerEventResources registers readable event MCP resources.
func registerEventResources(mcpServer *mcp.Server, client statev1.EventServiceClient) {
	mcpServer.AddResourceTemplate(domain.EventsListResourceTemplate(), domain.EventsListResourceHandler(client))
}

// registerContextResources registers readable context MCP resources.
func registerContextResources(mcpServer *mcp.Server, server *Server) {
	mcpServer.AddResource(domain.ContextResource(), domain.ContextResourceHandler(server.getContext))
}
