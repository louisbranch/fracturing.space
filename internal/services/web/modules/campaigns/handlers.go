package campaigns

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// DashboardSync exposes dashboard refresh hooks needed by campaign mutations.
type DashboardSync interface {
	CampaignCreated(context.Context, string, string)
	SessionStarted(context.Context, string, string)
	SessionEnded(context.Context, string, string)
	InviteChanged(context.Context, []string, string)
}

// campaignPageHandlerServices groups the shared workspace shell dependencies
// used by multiple campaign detail surfaces.
type campaignPageHandlerServices struct {
	workspace     campaignapp.CampaignWorkspaceService
	sessionReads  campaignapp.CampaignSessionReadService
	authorization campaignapp.CampaignAuthorizationService
}

// catalogHandlerServices groups campaign list and creation behavior.
type catalogHandlerServices struct {
	campaigns campaignapp.CampaignCatalogService
}

// starterHandlerServices groups protected starter preview and launch behavior.
type starterHandlerServices struct {
	starters campaignapp.CampaignStarterService
}

// overviewHandlerServices groups campaign overview, configuration, and AI
// binding behavior.
type overviewHandlerServices struct {
	automationReads  campaignapp.CampaignAutomationReadService
	automationMutate campaignapp.CampaignAutomationMutationService
	configuration    campaignapp.CampaignConfigurationService
}

// participantHandlerServices groups participant read and mutation behavior.
type participantHandlerServices struct {
	reads    campaignapp.CampaignParticipantReadService
	mutation campaignapp.CampaignParticipantMutationService
}

// characterHandlerServices groups character read, control, and mutation
// behavior.
type characterHandlerServices struct {
	reads    campaignapp.CampaignCharacterReadService
	control  campaignapp.CampaignCharacterControlService
	mutation campaignapp.CampaignCharacterMutationService
}

// creationHandlerServices groups character-creation workflow behavior.
type creationHandlerServices struct {
	pages    campaignworkflow.PageService
	mutation campaignworkflow.MutationService
}

// sessionHandlerServices groups session lifecycle behavior.
type sessionHandlerServices struct {
	mutation campaignapp.CampaignSessionMutationService
}

// inviteHandlerServices groups invite reads, mutations, and recipient lookup
// behavior.
type inviteHandlerServices struct {
	reads            campaignapp.CampaignInviteReadService
	mutation         campaignapp.CampaignInviteMutationService
	participantReads campaignapp.CampaignParticipantReadService
}

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	pages            campaignPageHandlerServices
	catalog          catalogHandlerServices
	starters         starterHandlerServices
	overview         overviewHandlerServices
	participants     participantHandlerServices
	characters       characterHandlerServices
	creation         creationHandlerServices
	sessions         sessionHandlerServices
	invites          inviteHandlerServices
	playFallbackPort string
	playLaunchGrant  playlaunchgrant.Config
	requestMeta      requestmeta.SchemePolicy
	nowFunc          func() time.Time
	sync             DashboardSync
}

// handlerServices groups the app-facing seams consumed by the transport layer.
type handlerServices struct {
	Page         campaignPageHandlerServices
	Catalog      catalogHandlerServices
	Starter      starterHandlerServices
	Overview     overviewHandlerServices
	Participants participantHandlerServices
	Characters   characterHandlerServices
	Creation     campaignCreationAppServices
	Sessions     sessionHandlerServices
	Invites      inviteHandlerServices
}

// campaignCreationAppServices keeps creation app-service assembly separate
// from workflow-owned transport services.
type campaignCreationAppServices struct {
	Pages campaignworkflow.PageAppService
	Flow  campaignworkflow.MutationAppService
}

// handlersConfig keeps the root transport constructor explicit by owned seam.
type handlersConfig struct {
	Services         handlerServices
	Base             modulehandler.Base
	PlayFallbackPort string
	PlayLaunchGrant  playlaunchgrant.Config
	RequestMeta      requestmeta.SchemePolicy
	Sync             DashboardSync
	Workflows        campaignworkflow.Registry
}

// newHandlerServices constructs the handler-facing capability bundle from
// route-surface-owned app configs.
func newHandlerServices(config serviceConfigs) handlerServices {
	workspace := campaignapp.NewWorkspaceService(config.Page.Workspace)
	authorization := campaignapp.NewAuthorizationService(config.Page.Authorization)
	sessionReads := campaignapp.NewSessionReadService(config.Page.SessionRead)
	participantReads := campaignapp.NewParticipantReadService(config.Participants.Read, config.Participants.Authorization)
	inviteParticipantReads := campaignapp.NewParticipantReadService(config.Invites.ParticipantRead, config.Invites.Authorization)
	creationPages := campaignworkflow.NewPageAppService(campaignapp.NewCharacterCreationPageService(config.Characters.Creation))
	creationFlow := campaignworkflow.NewMutationAppService(campaignapp.NewCharacterCreationMutationService(config.Characters.Creation, config.Characters.Authorization))
	return handlerServices{
		Page: campaignPageHandlerServices{
			workspace:     workspace,
			sessionReads:  sessionReads,
			authorization: authorization,
		},
		Catalog: catalogHandlerServices{
			campaigns: campaignapp.NewCatalogService(config.Catalog.Catalog),
		},
		Starter: starterHandlerServices{
			starters: campaignapp.NewStarterService(config.Starter.Starter),
		},
		Overview: overviewHandlerServices{
			automationReads:  campaignapp.NewAutomationReadService(config.Overview.AutomationRead, config.Overview.Authorization),
			automationMutate: campaignapp.NewAutomationMutationService(config.Overview.AutomationMutation, config.Overview.Authorization),
			configuration:    campaignapp.NewConfigurationService(config.Overview.Configuration, config.Overview.Authorization),
		},
		Participants: participantHandlerServices{
			reads:    participantReads,
			mutation: campaignapp.NewParticipantMutationService(config.Participants.Mutation, config.Participants.Authorization),
		},
		Characters: characterHandlerServices{
			reads:    campaignapp.NewCharacterReadService(config.Characters.Read, config.Characters.Authorization),
			control:  campaignapp.NewCharacterControlService(config.Characters.Control, config.Characters.Authorization),
			mutation: campaignapp.NewCharacterMutationService(config.Characters.Mutation, config.Characters.Authorization),
		},
		Creation: campaignCreationAppServices{
			Pages: creationPages,
			Flow:  creationFlow,
		},
		Sessions: sessionHandlerServices{
			mutation: campaignapp.NewSessionMutationService(config.Sessions.Mutation, config.Page.Authorization),
		},
		Invites: inviteHandlerServices{
			reads:            campaignapp.NewInviteReadService(config.Invites.Read, config.Invites.Authorization),
			mutation:         campaignapp.NewInviteMutationService(config.Invites.Mutation, config.Invites.Authorization),
			participantReads: inviteParticipantReads,
		},
	}
}

// newHandlers builds package wiring for this web seam from narrow app-facing contracts.
func newHandlers(config handlersConfig) handlers {
	var workflowMap campaignworkflow.Registry
	if config.Workflows != nil {
		workflowMap = config.Workflows
	}
	services := config.Services
	return handlers{
		Base:         config.Base,
		pages:        services.Page,
		catalog:      services.Catalog,
		starters:     services.Starter,
		overview:     services.Overview,
		participants: services.Participants,
		characters:   services.Characters,
		creation: creationHandlerServices{
			pages:    campaignworkflow.NewPageService(services.Creation.Pages, workflowMap),
			mutation: campaignworkflow.NewMutationService(services.Creation.Flow, workflowMap),
		},
		sessions: sessionHandlerServices{
			mutation: services.Sessions.mutation,
		},
		invites: inviteHandlerServices{
			reads:            services.Invites.reads,
			mutation:         services.Invites.mutation,
			participantReads: services.Invites.participantReads,
		},
		playFallbackPort: config.PlayFallbackPort,
		playLaunchGrant:  config.PlayLaunchGrant,
		requestMeta:      config.RequestMeta,
		nowFunc:          time.Now,
		sync:             config.Sync,
	}
}

// newHandlersFromConfig keeps production and test wiring convenient while
// routing transport ownership through narrower service groups.
func newHandlersFromConfig(
	config serviceConfigs,
	base modulehandler.Base,
	sync DashboardSync,
	workflows ...campaignworkflow.Registry,
) handlers {
	var workflowMap campaignworkflow.Registry
	if len(workflows) > 0 {
		workflowMap = workflows[0]
	}
	return newHandlers(handlersConfig{
		Services:  newHandlerServices(config),
		Base:      base,
		Sync:      sync,
		Workflows: workflowMap,
	})
}

// now centralizes this web behavior in one helper seam.
func (h handlers) now() time.Time {
	if h.nowFunc != nil {
		return h.nowFunc()
	}
	return time.Now()
}
