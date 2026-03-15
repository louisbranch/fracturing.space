package service

import (
	"fmt"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"
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
	getContext func() sessionctx.Context,
	notify sessionctx.ResourceUpdateNotifier,
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
		{tool: domain.CharacterCreationWorkflowApplyTool(), handler: domain.CharacterCreationWorkflowApplyHandler(characterClient, getContext, notify)},
		{tool: domain.CharacterStatePatchTool(), handler: domain.CharacterStatePatchHandler(snapshotClient, getContext, notify)},
	}
	for _, registration := range registrations {
		if err := registerTool(registrar, registration.tool, registration.handler); err != nil {
			return err
		}
	}
	return nil
}

func registerSessionTools(registrar mcpRegistrationTarget, client statev1.SessionServiceClient, getContext func() sessionctx.Context, notify sessionctx.ResourceUpdateNotifier) error {
	if err := registerTool(registrar, domain.SessionStartTool(), domain.SessionStartHandler(client, getContext, notify)); err != nil {
		return err
	}
	if err := registerTool(registrar, domain.SessionEndTool(), domain.SessionEndHandler(client, getContext, notify)); err != nil {
		return err
	}
	return nil
}

func registerSceneTools(registrar mcpRegistrationTarget, client statev1.SceneServiceClient, getContext func() sessionctx.Context, notify sessionctx.ResourceUpdateNotifier) error {
	return registerTool(registrar, domain.SceneCreateTool(), domain.SceneCreateHandler(client, getContext, notify))
}

func registerInternalAISceneTools(registrar mcpRegistrationTarget, client statev1.SceneServiceClient, getContext func() sessionctx.Context, notify sessionctx.ResourceUpdateNotifier) error {
	return registerSceneTools(registrar, client, getContext, notify)
}

func registerInteractionTools(registrar mcpRegistrationTarget, client statev1.InteractionServiceClient, getContext func() sessionctx.Context, notify sessionctx.ResourceUpdateNotifier) error {
	registrations := []struct {
		tool    *mcp.Tool
		handler any
	}{
		{tool: domain.InteractionSetActiveSceneTool(), handler: domain.InteractionSetActiveSceneHandler(client, getContext, notify)},
		{tool: domain.InteractionStartScenePlayerPhaseTool(), handler: domain.InteractionStartScenePlayerPhaseHandler(client, getContext, notify)},
		{tool: domain.InteractionSubmitScenePlayerPostTool(), handler: domain.InteractionSubmitScenePlayerPostHandler(client, getContext, notify)},
		{tool: domain.InteractionYieldScenePlayerPhaseTool(), handler: domain.InteractionYieldScenePlayerPhaseHandler(client, getContext, notify)},
		{tool: domain.InteractionUnyieldScenePlayerPhaseTool(), handler: domain.InteractionUnyieldScenePlayerPhaseHandler(client, getContext, notify)},
		{tool: domain.InteractionCommitSceneGMOutputTool(), handler: domain.InteractionCommitSceneGMOutputHandler(client, getContext, notify)},
		{tool: domain.InteractionAcceptScenePlayerPhaseTool(), handler: domain.InteractionAcceptScenePlayerPhaseHandler(client, getContext, notify)},
		{tool: domain.InteractionRequestScenePlayerRevisionsTool(), handler: domain.InteractionRequestScenePlayerRevisionsHandler(client, getContext, notify)},
		{tool: domain.InteractionEndScenePlayerPhaseTool(), handler: domain.InteractionEndScenePlayerPhaseHandler(client, getContext, notify)},
		{tool: domain.InteractionPauseOOCTool(), handler: domain.InteractionPauseOOCHandler(client, getContext, notify)},
		{tool: domain.InteractionPostOOCTool(), handler: domain.InteractionPostOOCHandler(client, getContext, notify)},
		{tool: domain.InteractionMarkOOCReadyTool(), handler: domain.InteractionMarkOOCReadyHandler(client, getContext, notify)},
		{tool: domain.InteractionClearOOCReadyTool(), handler: domain.InteractionClearOOCReadyHandler(client, getContext, notify)},
		{tool: domain.InteractionResumeOOCTool(), handler: domain.InteractionResumeOOCHandler(client, getContext, notify)},
	}
	for _, registration := range registrations {
		if err := registerTool(registrar, registration.tool, registration.handler); err != nil {
			return err
		}
	}
	return nil
}

func registerInternalAIInteractionTools(registrar mcpRegistrationTarget, client statev1.InteractionServiceClient, getContext func() sessionctx.Context, notify sessionctx.ResourceUpdateNotifier) error {
	registrations := []struct {
		tool    *mcp.Tool
		handler any
	}{
		{tool: domain.InteractionSetActiveSceneTool(), handler: domain.InteractionSetActiveSceneHandler(client, getContext, notify)},
		{tool: domain.InteractionStartScenePlayerPhaseTool(), handler: domain.InteractionStartScenePlayerPhaseHandler(client, getContext, notify)},
		{tool: domain.InteractionCommitSceneGMOutputTool(), handler: domain.InteractionCommitSceneGMOutputHandler(client, getContext, notify)},
		{tool: domain.InteractionAcceptScenePlayerPhaseTool(), handler: domain.InteractionAcceptScenePlayerPhaseHandler(client, getContext, notify)},
		{tool: domain.InteractionRequestScenePlayerRevisionsTool(), handler: domain.InteractionRequestScenePlayerRevisionsHandler(client, getContext, notify)},
		{tool: domain.InteractionEndScenePlayerPhaseTool(), handler: domain.InteractionEndScenePlayerPhaseHandler(client, getContext, notify)},
		{tool: domain.InteractionPauseOOCTool(), handler: domain.InteractionPauseOOCHandler(client, getContext, notify)},
		{tool: domain.InteractionPostOOCTool(), handler: domain.InteractionPostOOCHandler(client, getContext, notify)},
		{tool: domain.InteractionMarkOOCReadyTool(), handler: domain.InteractionMarkOOCReadyHandler(client, getContext, notify)},
		{tool: domain.InteractionClearOOCReadyTool(), handler: domain.InteractionClearOOCReadyHandler(client, getContext, notify)},
		{tool: domain.InteractionResumeOOCTool(), handler: domain.InteractionResumeOOCHandler(client, getContext, notify)},
	}
	for _, registration := range registrations {
		if err := registerTool(registrar, registration.tool, registration.handler); err != nil {
			return err
		}
	}
	return nil
}

func registerEventTools(registrar mcpRegistrationTarget, client statev1.EventServiceClient, getContext func() sessionctx.Context) error {
	return registerTool(registrar, domain.EventListTool(), domain.EventListHandler(client, getContext))
}

func registerForkTools(
	registrar mcpRegistrationTarget,
	client statev1.ForkServiceClient,
	getContext func() sessionctx.Context,
	notify sessionctx.ResourceUpdateNotifier,
) error {
	if err := registerTool(registrar, domain.CampaignForkTool(), domain.CampaignForkHandler(client, getContext, notify)); err != nil {
		return err
	}
	return registerTool(registrar, domain.CampaignLineageTool(), domain.CampaignLineageHandler(client, getContext))
}

// registerHarnessContextTools registers mutable context bootstrap tooling for
// non-production integration harnesses.
func registerHarnessContextTools(
	registrar mcpRegistrationTarget,
	_ statev1.CampaignServiceClient,
	_ statev1.SessionServiceClient,
	_ statev1.ParticipantServiceClient,
	server *Server,
	notify sessionctx.ResourceUpdateNotifier,
) error {
	// Harness bootstrap must set participant-scoped identity before normal game
	// reads are authorized, so this non-production path intentionally skips
	// gRPC existence validation and only mutates the in-process harness context.
	return registerTool(registrar, domain.SetContextTool(), domain.SetContextHandler(
		nil,
		nil,
		nil,
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
	getContext ...func() sessionctx.Context,
) {
	var contextGetter func() sessionctx.Context
	if len(getContext) != 0 {
		contextGetter = getContext[0]
	}
	registrar.AddResource(domain.CampaignListResource(), domain.CampaignListResourceHandler(campaignClient))
	registrar.AddResourceTemplate(domain.CampaignResourceTemplate(), domain.CampaignResourceHandler(campaignClient, contextGetter))
	registrar.AddResourceTemplate(domain.ParticipantListResourceTemplate(), domain.ParticipantListResourceHandler(participantClient, contextGetter))
	registrar.AddResourceTemplate(domain.CharacterListResourceTemplate(), domain.CharacterListResourceHandler(characterClient, contextGetter))
}

// registerSessionResources registers readable session MCP resources.
func registerSessionResources(registrar mcpRegistrationTarget, client statev1.SessionServiceClient, getContext ...func() sessionctx.Context) {
	var contextGetter func() sessionctx.Context
	if len(getContext) != 0 {
		contextGetter = getContext[0]
	}
	registrar.AddResourceTemplate(domain.SessionListResourceTemplate(), domain.SessionListResourceHandler(client, contextGetter))
}

// registerSceneResources registers readable scene MCP resources.
func registerSceneResources(registrar mcpRegistrationTarget, client statev1.SceneServiceClient, getContext ...func() sessionctx.Context) {
	var contextGetter func() sessionctx.Context
	if len(getContext) != 0 {
		contextGetter = getContext[0]
	}
	registrar.AddResourceTemplate(domain.SceneListResourceTemplate(), domain.SceneListResourceHandler(client, contextGetter))
}

// registerEventResources registers readable event MCP resources.
func registerEventResources(registrar mcpRegistrationTarget, client statev1.EventServiceClient, getContext ...func() sessionctx.Context) {
	var contextGetter func() sessionctx.Context
	if len(getContext) != 0 {
		contextGetter = getContext[0]
	}
	registrar.AddResourceTemplate(domain.EventsListResourceTemplate(), domain.EventsListResourceHandler(client, contextGetter))
}

// registerInteractionResources registers readable interaction MCP resources.
func registerInteractionResources(registrar mcpRegistrationTarget, client statev1.InteractionServiceClient, getContext ...func() sessionctx.Context) {
	var contextGetter func() sessionctx.Context
	if len(getContext) != 0 {
		contextGetter = getContext[0]
	}
	registrar.AddResourceTemplate(domain.InteractionStateResourceTemplate(), domain.InteractionStateResourceHandler(client, contextGetter))
}

// registerContextResources registers readable context MCP resources.
func registerContextResources(registrar mcpRegistrationTarget, server *Server) {
	registrar.AddResource(domain.ContextResource(), domain.ContextResourceHandler(server.getContext))
}

func registerInternalAICampaignResources(
	registrar mcpRegistrationTarget,
	campaignClient statev1.CampaignServiceClient,
	participantClient statev1.ParticipantServiceClient,
	characterClient statev1.CharacterServiceClient,
	getContext func() sessionctx.Context,
) {
	registrar.AddResourceTemplate(domain.CampaignResourceTemplate(), domain.CampaignResourceHandler(campaignClient, getContext))
	registrar.AddResourceTemplate(domain.ParticipantListResourceTemplate(), domain.ParticipantListResourceHandler(participantClient, getContext))
	registrar.AddResourceTemplate(domain.CharacterListResourceTemplate(), domain.CharacterListResourceHandler(characterClient, getContext))
}
