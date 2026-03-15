package service

import (
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	mcpcampaigncontext "github.com/louisbranch/fracturing.space/internal/services/mcp/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"
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

type mcpRegistrationProfile int

const (
	mcpRegistrationProfileStandard mcpRegistrationProfile = iota
	mcpRegistrationProfileHarness
	mcpRegistrationProfileInternalAI
)

const (
	mcpDaggerheartToolsModuleName    = "daggerheart-tools"
	mcpCampaignToolsModuleName       = "campaign-tools"
	mcpSessionToolsModuleName        = "session-tools"
	mcpSceneToolsModuleName          = "scene-tools"
	mcpInteractionToolsModuleName    = "interaction-tools"
	mcpForkToolsModuleName           = "fork-tools"
	mcpEventToolsModuleName          = "event-tools"
	mcpHarnessContextToolsModuleName = "harness-context-tools"
	mcpCampaignResourceModuleName    = "campaign-resources"
	mcpSessionResourceModuleName     = "session-resources"
	mcpSceneResourceModuleName       = "scene-resources"
	mcpInteractionResourceModuleName = "interaction-resources"
	mcpEventResourceModuleName       = "event-resources"
	mcpContextResourceModuleName     = "context-resources"
)

type mcpRegistrationClients struct {
	daggerheartClient      daggerheartv1.DaggerheartServiceClient
	campaignClient         statev1.CampaignServiceClient
	participantClient      statev1.ParticipantServiceClient
	characterClient        statev1.CharacterServiceClient
	snapshotClient         statev1.SnapshotServiceClient
	sessionClient          statev1.SessionServiceClient
	sceneClient            statev1.SceneServiceClient
	interactionClient      statev1.InteractionServiceClient
	forkClient             statev1.ForkServiceClient
	eventClient            statev1.EventServiceClient
	campaignArtifactClient aiv1.CampaignArtifactServiceClient
	systemReferenceClient  aiv1.SystemReferenceServiceClient
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
	newMCPToolRegistrar[domain.SceneCreateInput, domain.SceneCreateResult](),
	newMCPToolRegistrar[domain.InteractionSetActiveSceneInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.InteractionStartScenePlayerPhaseInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.InteractionSubmitScenePlayerPostInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.InteractionScenePhaseInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.InteractionCommitSceneGMOutputInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.InteractionAcceptScenePlayerPhaseInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.InteractionRequestScenePlayerRevisionsInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.InteractionEndScenePlayerPhaseInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.InteractionPauseOOCInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.InteractionPostOOCInput, domain.InteractionStateResult](),
	newMCPToolRegistrar[domain.EventListInput, domain.EventListResult](),
	newMCPToolRegistrar[domain.CampaignForkInput, domain.CampaignForkResult](),
	newMCPToolRegistrar[domain.CampaignLineageInput, domain.CampaignLineageResult](),
	newMCPToolRegistrar[domain.SetContextInput, domain.SetContextResult](),
	newMCPToolRegistrar[mcpcampaigncontext.ArtifactListInput, mcpcampaigncontext.ArtifactListResult](),
	newMCPToolRegistrar[mcpcampaigncontext.ArtifactGetInput, mcpcampaigncontext.ArtifactResult](),
	newMCPToolRegistrar[mcpcampaigncontext.ArtifactUpsertInput, mcpcampaigncontext.ArtifactResult](),
	newMCPToolRegistrar[mcpcampaigncontext.ReferenceSearchInput, mcpcampaigncontext.ReferenceSearchResult](),
	newMCPToolRegistrar[mcpcampaigncontext.ReferenceReadInput, mcpcampaigncontext.ReferenceDocumentResult](),
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
	args ...any,
) []mcpRegistrationModule {
	profile := mcpRegistrationProfileStandard
	var notify sessionctx.ResourceUpdateNotifier
	if len(args) != 0 {
		if value, ok := args[0].(mcpRegistrationProfile); ok {
			profile = value
		}
	}
	if len(args) > 1 {
		notify, _ = args[1].(sessionctx.ResourceUpdateNotifier)
	}
	if profile == mcpRegistrationProfileInternalAI {
		return newInternalAIRegistrationModules(server, clients, notify)
	}

	modules := []mcpRegistrationModule{
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
			name: mcpSceneToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerSceneTools(registrar, clients.sceneClient, server.getContext, notify)
			},
		},
		{
			name: mcpInteractionToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerInteractionTools(registrar, clients.interactionClient, server.getContext, notify)
			},
		},
		{
			name: mcpForkToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerForkTools(registrar, clients.forkClient, server.getContext, notify)
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
			name: mcpCampaignResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerCampaignResources(registrar, clients.campaignClient, clients.participantClient, clients.characterClient, server.getContext)
				return nil
			},
		},
		{
			name: mcpSessionResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerSessionResources(registrar, clients.sessionClient, server.getContext)
				return nil
			},
		},
		{
			name: mcpSceneResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerSceneResources(registrar, clients.sceneClient, server.getContext)
				return nil
			},
		},
		{
			name: mcpInteractionResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerInteractionResources(registrar, clients.interactionClient, server.getContext)
				return nil
			},
		},
		{
			name: mcpEventResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerEventResources(registrar, clients.eventClient, server.getContext)
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
	if profile == mcpRegistrationProfileHarness {
		modules = append(modules[:len(modules)-6], append([]mcpRegistrationModule{
			{
				name: mcpHarnessContextToolsModuleName,
				kind: mcpRegistrationKindTools,
				register: func(registrar mcpRegistrationTarget) error {
					return registerHarnessContextTools(registrar, clients.campaignClient, clients.sessionClient, clients.participantClient, server, notify)
				},
			},
		}, modules[len(modules)-6:]...)...)
	}
	if clients.campaignArtifactClient != nil || clients.systemReferenceClient != nil {
		modules = append(modules, mcpRegistrationModule{
			name: "campaign-context-tools",
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return mcpcampaigncontext.RegisterTools(registrar, clients.campaignArtifactClient, clients.systemReferenceClient, server.getContext, notify)
			},
		})
	}
	if clients.campaignArtifactClient != nil {
		modules = append(modules, mcpRegistrationModule{
			name: "campaign-context-resources",
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				mcpcampaigncontext.RegisterResources(registrar, clients.campaignArtifactClient)
				return nil
			},
		})
	}
	return modules
}

func newInternalAIRegistrationModules(
	server *Server,
	clients mcpRegistrationClients,
	notify sessionctx.ResourceUpdateNotifier,
) []mcpRegistrationModule {
	modules := []mcpRegistrationModule{
		{
			name: mcpDaggerheartToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerDaggerheartTools(registrar, clients.daggerheartClient)
			},
		},
		{
			name: mcpSceneToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerInternalAISceneTools(registrar, clients.sceneClient, server.getContext, notify)
			},
		},
		{
			name: mcpInteractionToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerInternalAIInteractionTools(registrar, clients.interactionClient, server.getContext, notify)
			},
		},
		{
			name: mcpCampaignResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerInternalAICampaignResources(registrar, clients.campaignClient, clients.participantClient, clients.characterClient, server.getContext)
				return nil
			},
		},
		{
			name: mcpSessionResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerSessionResources(registrar, clients.sessionClient, server.getContext)
				return nil
			},
		},
		{
			name: mcpSceneResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerSceneResources(registrar, clients.sceneClient, server.getContext)
				return nil
			},
		},
		{
			name: mcpInteractionResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerInteractionResources(registrar, clients.interactionClient, server.getContext)
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
	if clients.campaignArtifactClient != nil || clients.systemReferenceClient != nil {
		modules = append(modules, mcpRegistrationModule{
			name: "campaign-context-tools",
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return mcpcampaigncontext.RegisterTools(registrar, clients.campaignArtifactClient, clients.systemReferenceClient, server.getContext, notify)
			},
		})
	}
	if clients.campaignArtifactClient != nil {
		modules = append(modules, mcpRegistrationModule{
			name: "campaign-context-resources",
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				mcpcampaigncontext.RegisterResources(registrar, clients.campaignArtifactClient)
				return nil
			},
		})
	}
	return modules
}
