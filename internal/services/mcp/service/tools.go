package service

import (
	"fmt"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpRegistrationTarget interface {
	AddTool(*mcp.Tool, any) error
	AddResourceTemplate(*mcp.ResourceTemplate, mcp.ResourceHandler)
	AddResource(*mcp.Resource, mcp.ResourceHandler)
}

func registerDaggerheartTools(registrar mcpRegistrationTarget, client daggerheartv1.DaggerheartServiceClient) error {
	if err := registerTool(registrar, domain.ActionRollTool(), domain.ActionRollHandler(client)); err != nil {
		return err
	}
	if err := registerTool(registrar, domain.DualityExplainTool(), domain.DualityExplainHandler(client)); err != nil {
		return err
	}
	if err := registerTool(registrar, domain.DualityOutcomeTool(), domain.DualityOutcomeHandler(client)); err != nil {
		return err
	}
	if err := registerTool(registrar, domain.DualityProbabilityTool(), domain.DualityProbabilityHandler(client)); err != nil {
		return err
	}
	if err := registerTool(registrar, domain.RulesVersionTool(), domain.RulesVersionHandler(client)); err != nil {
		return err
	}
	if err := registerTool(registrar, domain.RollDiceTool(), domain.RollDiceHandler(client)); err != nil {
		return err
	}
	return nil
}

func registerCampaignTools(
	registrar mcpRegistrationTarget,
	campaignClient statev1.CampaignServiceClient,
	participantClient statev1.ParticipantServiceClient,
	characterClient statev1.CharacterServiceClient,
	snapshotClient statev1.SnapshotServiceClient,
	getContext func() domain.Context,
	notify domain.ResourceUpdateNotifier,
) error {
	registrations := []struct {
		tool    *mcp.Tool
		handler any
	}{
		{tool: domain.CampaignCreateTool(), handler: domain.CampaignCreateHandler(campaignClient, notify)},
		{tool: domain.CampaignEndTool(), handler: domain.CampaignEndHandler(campaignClient, getContext, notify)},
		{tool: domain.CampaignArchiveTool(), handler: domain.CampaignArchiveHandler(campaignClient, getContext, notify)},
		{tool: domain.CampaignRestoreTool(), handler: domain.CampaignRestoreHandler(campaignClient, getContext, notify)},
		{tool: domain.ParticipantCreateTool(), handler: domain.ParticipantCreateHandler(participantClient, getContext, notify)},
		{tool: domain.ParticipantUpdateTool(), handler: domain.ParticipantUpdateHandler(participantClient, getContext, notify)},
		{tool: domain.ParticipantDeleteTool(), handler: domain.ParticipantDeleteHandler(participantClient, getContext, notify)},
		{tool: domain.CharacterCreateTool(), handler: domain.CharacterCreateHandler(characterClient, getContext, notify)},
		{tool: domain.CharacterUpdateTool(), handler: domain.CharacterUpdateHandler(characterClient, getContext, notify)},
		{tool: domain.CharacterDeleteTool(), handler: domain.CharacterDeleteHandler(characterClient, getContext, notify)},
		{tool: domain.CharacterControlSetTool(), handler: domain.CharacterControlSetHandler(characterClient, getContext, notify)},
		{tool: domain.CharacterSheetGetTool(), handler: domain.CharacterSheetGetHandler(characterClient, getContext)},
		{tool: domain.CharacterProfilePatchTool(), handler: domain.CharacterProfilePatchHandler(characterClient, getContext, notify)},
		{tool: domain.CharacterStatePatchTool(), handler: domain.CharacterStatePatchHandler(snapshotClient, getContext, notify)},
	}
	for _, registration := range registrations {
		if err := registerTool(registrar, registration.tool, registration.handler); err != nil {
			return err
		}
	}
	return nil
}

func registerSessionTools(registrar mcpRegistrationTarget, client statev1.SessionServiceClient, getContext func() domain.Context, notify domain.ResourceUpdateNotifier) error {
	if err := registerTool(registrar, domain.SessionStartTool(), domain.SessionStartHandler(client, getContext, notify)); err != nil {
		return err
	}
	if err := registerTool(registrar, domain.SessionEndTool(), domain.SessionEndHandler(client, getContext, notify)); err != nil {
		return err
	}
	return nil
}

func registerEventTools(registrar mcpRegistrationTarget, client statev1.EventServiceClient, getContext func() domain.Context) error {
	return registerTool(registrar, domain.EventListTool(), domain.EventListHandler(client, getContext))
}

func registerForkTools(registrar mcpRegistrationTarget, client statev1.ForkServiceClient, notify domain.ResourceUpdateNotifier) error {
	if err := registerTool(registrar, domain.CampaignForkTool(), domain.CampaignForkHandler(client, notify)); err != nil {
		return err
	}
	return registerTool(registrar, domain.CampaignLineageTool(), domain.CampaignLineageHandler(client))
}

// registerContextTools registers context management tools.
func registerContextTools(
	registrar mcpRegistrationTarget,
	campaignClient statev1.CampaignServiceClient,
	sessionClient statev1.SessionServiceClient,
	participantClient statev1.ParticipantServiceClient,
	server *Server,
	notify domain.ResourceUpdateNotifier,
) error {
	return registerTool(registrar, domain.SetContextTool(), domain.SetContextHandler(
		campaignClient,
		sessionClient,
		participantClient,
		server.setContext,
		server.getContext,
		notify,
	))
}

func registerTool(registrar mcpRegistrationTarget, tool *mcp.Tool, handler any) error {
	if tool == nil {
		return fmt.Errorf("tool is nil")
	}
	return registrar.AddTool(tool, handler)
}

// registerCampaignResources registers readable campaign MCP resources.
func registerCampaignResources(
	registrar mcpRegistrationTarget,
	campaignClient statev1.CampaignServiceClient,
	participantClient statev1.ParticipantServiceClient,
	characterClient statev1.CharacterServiceClient,
) {
	registrar.AddResource(domain.CampaignListResource(), domain.CampaignListResourceHandler(campaignClient))
	registrar.AddResourceTemplate(domain.CampaignResourceTemplate(), domain.CampaignResourceHandler(campaignClient))
	registrar.AddResourceTemplate(domain.ParticipantListResourceTemplate(), domain.ParticipantListResourceHandler(participantClient))
	registrar.AddResourceTemplate(domain.CharacterListResourceTemplate(), domain.CharacterListResourceHandler(characterClient))
}

// registerSessionResources registers readable session MCP resources.
func registerSessionResources(registrar mcpRegistrationTarget, client statev1.SessionServiceClient) {
	registrar.AddResourceTemplate(domain.SessionListResourceTemplate(), domain.SessionListResourceHandler(client))
}

// registerEventResources registers readable event MCP resources.
func registerEventResources(registrar mcpRegistrationTarget, client statev1.EventServiceClient) {
	registrar.AddResourceTemplate(domain.EventsListResourceTemplate(), domain.EventsListResourceHandler(client))
}

// registerContextResources registers readable context MCP resources.
func registerContextResources(registrar mcpRegistrationTarget, server *Server) {
	registrar.AddResource(domain.ContextResource(), domain.ContextResourceHandler(server.getContext))
}
