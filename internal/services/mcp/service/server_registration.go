package service

import (
	"fmt"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpRegistrationKind int

const (
	mcpRegistrationKindTools mcpRegistrationKind = iota
	mcpRegistrationKindResources
)

type mcpRegistrationModule struct {
	name     string
	kind     mcpRegistrationKind
	register func(mcpRegistrationTarget) error
}

const (
	mcpDaggerheartToolsModuleName = "daggerheart-tools"
	mcpCampaignToolsModuleName    = "campaign-tools"
	mcpSessionToolsModuleName     = "session-tools"
	mcpForkToolsModuleName        = "fork-tools"
	mcpEventToolsModuleName       = "event-tools"
	mcpContextToolsModuleName     = "context-tools"
	mcpCampaignResourceModuleName = "campaign-resources"
	mcpSessionResourceModuleName  = "session-resources"
	mcpEventResourceModuleName    = "event-resources"
	mcpContextResourceModuleName  = "context-resources"
)

type mcpRegistrationClients struct {
	daggerheartClient daggerheartv1.DaggerheartServiceClient
	campaignClient    statev1.CampaignServiceClient
	participantClient statev1.ParticipantServiceClient
	characterClient   statev1.CharacterServiceClient
	snapshotClient    statev1.SnapshotServiceClient
	sessionClient     statev1.SessionServiceClient
	forkClient        statev1.ForkServiceClient
	eventClient       statev1.EventServiceClient
}

type mcpServerRegistrationAdapter struct {
	server *mcp.Server
}

func (r mcpServerRegistrationAdapter) AddTool(tool *mcp.Tool, handler any) error {
	return addMCPTool(r.server, tool, handler)
}

func (r mcpServerRegistrationAdapter) AddResourceTemplate(resourceTemplate *mcp.ResourceTemplate, handler mcp.ResourceHandler) {
	r.server.AddResourceTemplate(resourceTemplate, handler)
}

func (r mcpServerRegistrationAdapter) AddResource(resource *mcp.Resource, handler mcp.ResourceHandler) {
	r.server.AddResource(resource, handler)
}

type mcpToolRegistrar struct {
	matches func(any) bool
	add     func(*mcp.Server, *mcp.Tool, any)
}

func newMCPToolRegistrar[I any, O any]() mcpToolRegistrar {
	return mcpToolRegistrar{
		matches: func(handler any) bool {
			_, ok := handler.(mcp.ToolHandlerFor[I, O])
			return ok
		},
		add: func(server *mcp.Server, tool *mcp.Tool, handler any) {
			mcp.AddTool(server, tool, handler.(mcp.ToolHandlerFor[I, O]))
		},
	}
}

var mcpToolRegistrars = []mcpToolRegistrar{
	newMCPToolRegistrar[domain.ActionRollInput, domain.ActionRollResult](),
	newMCPToolRegistrar[domain.DualityExplainInput, domain.DualityExplainResult](),
	newMCPToolRegistrar[domain.DualityOutcomeInput, domain.DualityOutcomeResult](),
	newMCPToolRegistrar[domain.DualityProbabilityInput, domain.DualityProbabilityResult](),
	newMCPToolRegistrar[domain.RulesVersionInput, domain.RulesVersionResult](),
	newMCPToolRegistrar[domain.RollDiceInput, domain.RollDiceResult](),
	newMCPToolRegistrar[domain.CampaignCreateInput, domain.CampaignCreateResult](),
	newMCPToolRegistrar[domain.CampaignStatusChangeInput, domain.CampaignStatusResult](),
	newMCPToolRegistrar[domain.ParticipantCreateInput, domain.ParticipantCreateResult](),
	newMCPToolRegistrar[domain.ParticipantUpdateInput, domain.ParticipantUpdateResult](),
	newMCPToolRegistrar[domain.ParticipantDeleteInput, domain.ParticipantDeleteResult](),
	newMCPToolRegistrar[domain.CharacterCreateInput, domain.CharacterCreateResult](),
	newMCPToolRegistrar[domain.CharacterUpdateInput, domain.CharacterUpdateResult](),
	newMCPToolRegistrar[domain.CharacterDeleteInput, domain.CharacterDeleteResult](),
	newMCPToolRegistrar[domain.CharacterControlSetInput, domain.CharacterControlSetResult](),
	newMCPToolRegistrar[domain.CharacterSheetGetInput, domain.CharacterSheetGetResult](),
	newMCPToolRegistrar[domain.CharacterProfilePatchInput, domain.CharacterProfilePatchResult](),
	newMCPToolRegistrar[domain.CharacterCreationWorkflowApplyInput, domain.CharacterCreationWorkflowApplyResult](),
	newMCPToolRegistrar[domain.CharacterStatePatchInput, domain.CharacterStatePatchResult](),
	newMCPToolRegistrar[domain.SessionStartInput, domain.SessionStartResult](),
	newMCPToolRegistrar[domain.SessionEndInput, domain.SessionEndResult](),
	newMCPToolRegistrar[domain.EventListInput, domain.EventListResult](),
	newMCPToolRegistrar[domain.CampaignForkInput, domain.CampaignForkResult](),
	newMCPToolRegistrar[domain.CampaignLineageInput, domain.CampaignLineageResult](),
	newMCPToolRegistrar[domain.SetContextInput, domain.SetContextResult](),
}

func addMCPTool(server *mcp.Server, tool *mcp.Tool, handler any) error {
	for _, registrar := range mcpToolRegistrars {
		if registrar.matches(handler) {
			registrar.add(server, tool, handler)
			return nil
		}
	}
	toolName := "<nil>"
	if tool != nil {
		toolName = tool.Name
	}
	return fmt.Errorf("mcp registration adapter does not support handler type %T for tool %q", handler, toolName)
}

func newMCPRegistrationModules(
	server *Server,
	clients mcpRegistrationClients,
	notify domain.ResourceUpdateNotifier,
) []mcpRegistrationModule {
	return []mcpRegistrationModule{
		{
			name: mcpDaggerheartToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerDaggerheartTools(registrar, clients.daggerheartClient)
			},
		},
		{
			name: mcpCampaignToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerCampaignTools(registrar, clients.campaignClient, clients.participantClient, clients.characterClient, clients.snapshotClient, server.getContext, notify)
			},
		},
		{
			name: mcpSessionToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerSessionTools(registrar, clients.sessionClient, server.getContext, notify)
			},
		},
		{
			name: mcpForkToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerForkTools(registrar, clients.forkClient, notify)
			},
		},
		{
			name: mcpEventToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerEventTools(registrar, clients.eventClient, server.getContext)
			},
		},
		{
			name: mcpContextToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerContextTools(registrar, clients.campaignClient, clients.sessionClient, clients.participantClient, server, notify)
			},
		},
		{
			name: mcpCampaignResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerCampaignResources(registrar, clients.campaignClient, clients.participantClient, clients.characterClient)
				return nil
			},
		},
		{
			name: mcpSessionResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerSessionResources(registrar, clients.sessionClient)
				return nil
			},
		},
		{
			name: mcpEventResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerEventResources(registrar, clients.eventClient)
				return nil
			},
		},
		{
			name: mcpContextResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerContextResources(registrar, server)
				return nil
			},
		},
	}
}
