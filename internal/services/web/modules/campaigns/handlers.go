package campaigns

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

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

// newProductionHandlerServices constructs the handler-facing capability bundle
// directly from production composition inputs so contributors can trace one
// surface without hopping through an extra root config aggregate.
func newProductionHandlerServices(config CompositionConfig) handlerServices {
	characterSurface := newCharacterSurfaceConfig(config)
	return handlerServices{
		Page:         newCampaignPageHandlerServices(newPageServiceConfig(config)),
		Catalog:      newCatalogHandlerServices(newCatalogSurfaceConfig(config)),
		Starter:      newStarterHandlerServices(newStarterSurfaceConfig(config)),
		Overview:     newOverviewHandlerServices(newOverviewSurfaceConfig(config)),
		Participants: newParticipantHandlerServices(newParticipantSurfaceConfig(config)),
		Characters:   newCharacterHandlerServices(characterSurface),
		Creation:     newCampaignCreationAppServices(characterSurface),
		Sessions:     newSessionHandlerServices(newSessionSurfaceConfig(config), newPageAuthorizationGateway(config)),
		Invites:      newInviteHandlerServices(newInviteSurfaceConfig(config)),
	}
}

// newHandlers builds package wiring for this web seam from narrow app-facing contracts.
func newHandlers(config handlersConfig) (handlers, error) {
	services := config.Services
	missing := missingHandlerServices(services)
	if len(missing) > 0 {
		return handlers{}, fmt.Errorf("campaigns module missing required services: %s", strings.Join(missing, ", "))
	}

	support := newCampaignRouteSupport(config.Base, config.RequestMeta, config.Sync)
	detail := newCampaignDetailHandlers(support, services.Page)
	creation := newCreationHandlerServices(services.Creation, config.Systems.workflowRegistry())

	return handlers{
		catalog:      newCatalogHandlers(support, services.Catalog, config.Systems),
		starters:     newStarterHandlers(support, services.Starter),
		overview:     newOverviewHandlers(detail, services.Overview),
		participants: newParticipantHandlers(detail, services.Participants),
		characters:   newCharacterHandlers(detail, services.Characters, creation),
		creation:     newStandaloneCreationHandlers(detail, creation),
		sessions:     newSessionHandlers(detail, services.Sessions, config.PlayFallbackPort, config.PlayLaunchGrant),
		invites:      newInviteHandlers(detail, services.Invites),
	}, nil
}

// missingHandlerServices reports the owned handler seams that were not wired so
// constructor failures can name the broken campaign surface directly.
func missingHandlerServices(services handlerServices) []string {
	missing := []string{}
	missing = append(missing, missingCampaignPageHandlerServices(services.Page)...)
	missing = append(missing, missingCatalogHandlerServices(services.Catalog)...)
	missing = append(missing, missingOverviewHandlerServices(services.Overview)...)
	missing = append(missing, missingParticipantHandlerServices(services.Participants)...)
	missing = append(missing, missingCharacterHandlerServices(services.Characters)...)
	missing = append(missing, missingCampaignCreationAppServices(services.Creation)...)
	missing = append(missing, missingSessionHandlerServices(services.Sessions)...)
	missing = append(missing, missingInviteHandlerServices(services.Invites)...)
	return missing
}
