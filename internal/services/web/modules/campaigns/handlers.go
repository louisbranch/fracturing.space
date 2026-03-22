package campaigns

import (
	"errors"
	"fmt"

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
// directly from production composition inputs. Each surface validates its own
// dependencies at construction time and returns an error on missing gateways.
func newProductionHandlerServices(config CompositionConfig) (handlerServices, error) {
	var errs []error

	page, err := newCampaignPageHandlerServices(newPageServiceConfig(config))
	if err != nil {
		errs = append(errs, err)
	}
	catalog, err := newCatalogHandlerServices(newCatalogSurfaceConfig(config))
	if err != nil {
		errs = append(errs, err)
	}
	starter, err := newStarterHandlerServices(newStarterSurfaceConfig(config))
	if err != nil {
		errs = append(errs, err)
	}
	overview, err := newOverviewHandlerServices(newOverviewSurfaceConfig(config))
	if err != nil {
		errs = append(errs, err)
	}
	participants, err := newParticipantHandlerServices(newParticipantSurfaceConfig(config))
	if err != nil {
		errs = append(errs, err)
	}
	characterSurface := newCharacterSurfaceConfig(config)
	characters, err := newCharacterHandlerServices(characterSurface)
	if err != nil {
		errs = append(errs, err)
	}
	creation, err := newCampaignCreationAppServices(characterSurface)
	if err != nil {
		errs = append(errs, err)
	}
	sessions, err := newSessionHandlerServices(newSessionSurfaceConfig(config), newPageAuthorizationGateway(config))
	if err != nil {
		errs = append(errs, err)
	}
	invites, err := newInviteHandlerServices(newInviteSurfaceConfig(config))
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return handlerServices{}, fmt.Errorf("campaigns module missing required services: %w", errors.Join(errs...))
	}
	return handlerServices{
		Page:         page,
		Catalog:      catalog,
		Starter:      starter,
		Overview:     overview,
		Participants: participants,
		Characters:   characters,
		Creation:     creation,
		Sessions:     sessions,
		Invites:      invites,
	}, nil
}

// newHandlers builds package wiring for this web seam from narrow app-facing
// contracts. Returns an error when core services are absent — this catches
// callers that skip the builder-level validation in newProductionHandlerServices.
func newHandlers(config handlersConfig) (handlers, error) {
	services := config.Services
	if services.Page.workspace == nil || services.Catalog.campaigns == nil {
		return handlers{}, fmt.Errorf("campaigns module missing required services")
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
