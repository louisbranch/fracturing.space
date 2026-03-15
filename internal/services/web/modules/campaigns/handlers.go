package campaigns

import (
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// DashboardSync keeps campaign mutations aligned with dashboard freshness.
type DashboardSync = dashboardsync.Service

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
	catalog      catalogHandlers
	starters     starterHandlers
	overview     overviewHandlers
	participants participantHandlers
	characters   characterHandlers
	creation     creationHandlers
	sessions     sessionHandlers
	invites      inviteHandlers
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
	Systems          campaignSystemRegistry
}

// catalogHandlers owns the campaign catalog and creation transport surface.
type catalogHandlers struct {
	campaignRouteSupport
	catalog catalogHandlerServices
	systems campaignSystemRegistry
}

// starterHandlers owns the protected starter preview and launch surface.
type starterHandlers struct {
	campaignRouteSupport
	starters starterHandlerServices
}

// campaignDetailHandlers owns the shared workspace-shell route support used by
// detail, creation, session, and invite surfaces.
type campaignDetailHandlers struct {
	campaignRouteSupport
	pages campaignPageHandlerServices
}

// overviewHandlers owns overview/configuration/AI-binding routes.
type overviewHandlers struct {
	campaignDetailHandlers
	overview overviewHandlerServices
}

// participantHandlers owns participant read and mutation routes.
type participantHandlers struct {
	campaignDetailHandlers
	participants participantHandlerServices
}

// characterHandlers owns character read, control, and mutation routes.
type characterHandlers struct {
	campaignDetailHandlers
	characters characterHandlerServices
	creation   creationHandlerServices
}

// creationHandlers owns character-creation page and workflow routes.
type creationHandlers struct {
	campaignDetailHandlers
	creation creationHandlerServices
}

// sessionHandlers owns session detail/lifecycle plus play-launch routes.
type sessionHandlers struct {
	campaignDetailHandlers
	sessions         sessionHandlerServices
	playFallbackPort string
	playLaunchGrant  playlaunchgrant.Config
}

// inviteHandlers owns invite read, search, and mutation routes.
type inviteHandlers struct {
	campaignDetailHandlers
	invites inviteHandlerServices
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
func newHandlers(config handlersConfig) (handlers, error) {
	services := config.Services
	missing := missingHandlerServices(services)
	if len(missing) > 0 {
		return handlers{}, fmt.Errorf("campaigns module missing required services: %s", strings.Join(missing, ", "))
	}
	sync := config.Sync
	if sync == nil {
		sync = dashboardsync.Noop{}
	}
	support := campaignRouteSupport{
		Base:        config.Base,
		requestMeta: config.RequestMeta,
		nowFunc:     time.Now,
		sync:        sync,
	}
	detail := campaignDetailHandlers{
		campaignRouteSupport: support,
		pages:                services.Page,
	}
	creation := creationHandlerServices{
		pages:    campaignworkflow.NewPageService(services.Creation.Pages, config.Systems.workflowRegistry()),
		mutation: campaignworkflow.NewMutationService(services.Creation.Flow, config.Systems.workflowRegistry()),
	}
	return handlers{
		catalog: catalogHandlers{
			campaignRouteSupport: support,
			catalog:              services.Catalog,
			systems:              config.Systems,
		},
		starters: starterHandlers{
			campaignRouteSupport: support,
			starters:             services.Starter,
		},
		overview: overviewHandlers{
			campaignDetailHandlers: detail,
			overview:               services.Overview,
		},
		participants: participantHandlers{
			campaignDetailHandlers: detail,
			participants:           services.Participants,
		},
		characters: characterHandlers{
			campaignDetailHandlers: detail,
			characters:             services.Characters,
			creation:               creation,
		},
		creation: creationHandlers{
			campaignDetailHandlers: detail,
			creation:               creation,
		},
		sessions: sessionHandlers{
			campaignDetailHandlers: detail,
			sessions:               services.Sessions,
			playFallbackPort:       config.PlayFallbackPort,
			playLaunchGrant:        config.PlayLaunchGrant,
		},
		invites: inviteHandlers{
			campaignDetailHandlers: detail,
			invites:                services.Invites,
		},
	}, nil
}

// newHandlersFromConfig keeps production and test wiring convenient while
// routing transport ownership through narrower service groups.
func newHandlersFromConfig(
	config serviceConfigs,
	base modulehandler.Base,
	sync DashboardSync,
	workflows ...campaignworkflow.Registry,
) handlers {
	handlerSet, err := newHandlers(handlersConfig{
		Services: newHandlerServices(config),
		Base:     base,
		Sync:     sync,
		Systems:  newCampaignSystemsFromWorkflows(workflows...),
	})
	if err != nil {
		panic(err)
	}
	return handlerSet
}

// missingHandlerServices reports the owned handler seams that were not wired so
// constructor failures can name the broken campaign surface directly.
func missingHandlerServices(services handlerServices) []string {
	missing := []string{}
	if services.Page.workspace == nil {
		missing = append(missing, "page-workspace")
	}
	if services.Page.sessionReads == nil {
		missing = append(missing, "page-sessions")
	}
	if services.Page.authorization == nil {
		missing = append(missing, "page-authorization")
	}
	if services.Catalog.campaigns == nil {
		missing = append(missing, "catalog")
	}
	if services.Overview.automationReads == nil {
		missing = append(missing, "overview-automation-reads")
	}
	if services.Overview.automationMutate == nil {
		missing = append(missing, "overview-automation-mutation")
	}
	if services.Overview.configuration == nil {
		missing = append(missing, "overview-configuration")
	}
	if services.Participants.reads == nil {
		missing = append(missing, "participant-reads")
	}
	if services.Participants.mutation == nil {
		missing = append(missing, "participant-mutation")
	}
	if services.Characters.reads == nil {
		missing = append(missing, "character-reads")
	}
	if services.Characters.control == nil {
		missing = append(missing, "character-control")
	}
	if services.Characters.mutation == nil {
		missing = append(missing, "character-mutation")
	}
	if services.Creation.Pages == nil {
		missing = append(missing, "creation-pages")
	}
	if services.Creation.Flow == nil {
		missing = append(missing, "creation-flow")
	}
	if services.Sessions.mutation == nil {
		missing = append(missing, "session-mutation")
	}
	if services.Invites.reads == nil {
		missing = append(missing, "invite-reads")
	}
	if services.Invites.mutation == nil {
		missing = append(missing, "invite-mutation")
	}
	if services.Invites.participantReads == nil {
		missing = append(missing, "invite-participant-reads")
	}
	return missing
}
