package campaigns

import (
	"context"
	"time"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// DashboardSync exposes dashboard refresh hooks needed by campaign mutations.
type DashboardSync interface {
	CampaignCreated(context.Context, string, string)
	SessionStarted(context.Context, string, string)
	SessionEnded(context.Context, string, string)
	InviteChanged(context.Context, []string, string)
}

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	catalog           campaignapp.CampaignCatalogService
	workspace         campaignapp.CampaignWorkspaceService
	game              campaignapp.CampaignGameService
	participantReads  campaignapp.CampaignParticipantReadService
	participantMutate campaignapp.CampaignParticipantMutationService
	automationReads   campaignapp.CampaignAutomationReadService
	automationMutate  campaignapp.CampaignAutomationMutationService
	characterReads    campaignapp.CampaignCharacterReadService
	characterControl  campaignapp.CampaignCharacterControlService
	characterMutate   campaignapp.CampaignCharacterMutationService
	sessionReads      campaignapp.CampaignSessionReadService
	sessionMutate     campaignapp.CampaignSessionMutationService
	inviteReads       campaignapp.CampaignInviteReadService
	inviteMutate      campaignapp.CampaignInviteMutationService
	configuration     campaignapp.CampaignConfigurationService
	authorization     campaignapp.CampaignAuthorizationService
	creationPages     campaignworkflow.PageService
	creationMutation  campaignworkflow.MutationService
	chatFallbackPort  string
	nowFunc           func() time.Time
	sync              DashboardSync
}

// handlerServices groups the app-facing seams consumed by the transport layer.
type handlerServices struct {
	Catalog            campaignapp.CampaignCatalogService
	Workspace          campaignapp.CampaignWorkspaceService
	Game               campaignapp.CampaignGameService
	ParticipantReads   campaignapp.CampaignParticipantReadService
	ParticipantMutate  campaignapp.CampaignParticipantMutationService
	AutomationReads    campaignapp.CampaignAutomationReadService
	AutomationMutation campaignapp.CampaignAutomationMutationService
	CharacterReads     campaignapp.CampaignCharacterReadService
	CharacterControl   campaignapp.CampaignCharacterControlService
	CharacterMutation  campaignapp.CampaignCharacterMutationService
	SessionReads       campaignapp.CampaignSessionReadService
	SessionMutation    campaignapp.CampaignSessionMutationService
	InviteReads        campaignapp.CampaignInviteReadService
	InviteMutation     campaignapp.CampaignInviteMutationService
	Configuration      campaignapp.CampaignConfigurationService
	Authorization      campaignapp.CampaignAuthorizationService
	CreationPages      campaignworkflow.PageAppService
	CreationFlow       campaignworkflow.MutationAppService
}

// handlersConfig keeps the root transport constructor explicit by owned seam.
type handlersConfig struct {
	Services         handlerServices
	Base             modulehandler.Base
	ChatFallbackPort string
	Sync             DashboardSync
	Workflows        campaignworkflow.Registry
}

// newHandlerServices constructs the handler-facing capability bundle from
// explicit app service constructors.
func newHandlerServices(config campaignapp.ServiceConfig) handlerServices {
	return handlerServices{
		Catalog:            campaignapp.NewCatalogService(config.Catalog),
		Workspace:          campaignapp.NewWorkspaceService(config.Workspace),
		Game:               campaignapp.NewGameService(config.Game),
		ParticipantReads:   campaignapp.NewParticipantReadService(config.ParticipantRead, config.Authorization),
		ParticipantMutate:  campaignapp.NewParticipantMutationService(config.ParticipantMutation, config.Authorization),
		AutomationReads:    campaignapp.NewAutomationReadService(config.AutomationRead, config.Authorization),
		AutomationMutation: campaignapp.NewAutomationMutationService(config.AutomationMutation, config.Authorization),
		CharacterReads:     campaignapp.NewCharacterReadService(config.CharacterRead, config.Authorization),
		CharacterControl:   campaignapp.NewCharacterControlService(config.CharacterControl, config.Authorization),
		CharacterMutation:  campaignapp.NewCharacterMutationService(config.CharacterMutation, config.Authorization),
		SessionReads:       campaignapp.NewSessionReadService(config.SessionRead),
		SessionMutation:    campaignapp.NewSessionMutationService(config.SessionMutation, config.Authorization),
		InviteReads:        campaignapp.NewInviteReadService(config.InviteRead, config.Authorization),
		InviteMutation:     campaignapp.NewInviteMutationService(config.InviteMutation, config.Authorization),
		Configuration:      campaignapp.NewConfigurationService(config.Configuration, config.Authorization),
		Authorization:      campaignapp.NewAuthorizationService(config.Authorization),
		CreationPages:      campaignworkflow.NewPageAppService(campaignapp.NewCharacterCreationPageService(config.Creation)),
		CreationFlow:       campaignworkflow.NewMutationAppService(campaignapp.NewCharacterCreationMutationService(config.Creation, config.Authorization)),
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
		Base:              config.Base,
		catalog:           services.Catalog,
		workspace:         services.Workspace,
		game:              services.Game,
		participantReads:  services.ParticipantReads,
		participantMutate: services.ParticipantMutate,
		automationReads:   services.AutomationReads,
		automationMutate:  services.AutomationMutation,
		characterReads:    services.CharacterReads,
		characterControl:  services.CharacterControl,
		characterMutate:   services.CharacterMutation,
		sessionReads:      services.SessionReads,
		sessionMutate:     services.SessionMutation,
		inviteReads:       services.InviteReads,
		inviteMutate:      services.InviteMutation,
		configuration:     services.Configuration,
		authorization:     services.Authorization,
		creationPages:     campaignworkflow.NewPageService(services.CreationPages, workflowMap),
		creationMutation:  campaignworkflow.NewMutationService(services.CreationFlow, workflowMap),
		chatFallbackPort:  config.ChatFallbackPort,
		nowFunc:           time.Now,
		sync:              config.Sync,
	}
}

// newHandlersFromConfig keeps production and test wiring convenient while
// routing transport ownership through narrower service groups.
func newHandlersFromConfig(
	serviceConfig campaignapp.ServiceConfig,
	base modulehandler.Base,
	chatFallbackPort string,
	sync DashboardSync,
	workflows ...campaignworkflow.Registry,
) handlers {
	var workflowMap campaignworkflow.Registry
	if len(workflows) > 0 {
		workflowMap = workflows[0]
	}
	return newHandlers(handlersConfig{
		Services:         newHandlerServices(serviceConfig),
		Base:             base,
		ChatFallbackPort: chatFallbackPort,
		Sync:             sync,
		Workflows:        workflowMap,
	})
}

// now centralizes this web behavior in one helper seam.
func (h handlers) now() time.Time {
	if h.nowFunc != nil {
		return h.nowFunc()
	}
	return time.Now()
}
