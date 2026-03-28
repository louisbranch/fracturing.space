package campaigns

import (
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	campaigncharacters "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/characters"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaigninvites "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/invites"
	campaignoverview "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/overview"
	campaignparticipants "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/participants"
	campaignsessions "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/sessions"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	catalog      catalogHandlers
	starters     starterHandlers
	overview     campaignoverview.Handler
	participants campaignparticipants.Handler
	characters   campaigncharacters.Handler
	sessions     campaignsessions.Handler
	invites      campaigninvites.Handler
}

// handlerServices groups the app-facing seams consumed by the transport layer.
type handlerServices struct {
	Page         campaigndetail.PageServices
	Catalog      catalogHandlerServices
	Starter      starterHandlerServices
	Overview     campaignoverview.HandlerServices
	Participants campaignparticipants.HandlerServices
	Characters   campaigncharacters.HandlerServices
	Sessions     campaignsessions.HandlerServices
	Invites      campaigninvites.HandlerServices
}

// handlersConfig keeps the root transport constructor explicit by owned seam.
type handlersConfig struct {
	Services         handlerServices
	Base             modulehandler.Base
	PlayFallbackPort string
	PlayLaunchGrant  playlaunchgrant.Config
	RequestMeta      requestmeta.SchemePolicy
	Sync             campaigndetail.DashboardSync
	Systems          campaignSystemRegistry
}

// newProductionHandlerServices constructs the handler-facing capability bundle
// directly from production composition inputs. Each surface validates its own
// dependencies at construction time and returns an error on missing gateways.
func newProductionHandlerServices(config CompositionConfig) (handlerServices, error) {
	var errs []error

	page, err := campaigndetail.NewPageServices(newPageServiceConfig(config))
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
	overview, err := campaignoverview.NewHandlerServices(newOverviewSurfaceConfig(config))
	if err != nil {
		errs = append(errs, err)
	}
	participants, err := campaignparticipants.NewHandlerServices(newParticipantSurfaceConfig(config))
	if err != nil {
		errs = append(errs, err)
	}
	characterSurface := newCharacterSurfaceConfig(config)
	characters, err := campaigncharacters.NewHandlerServices(characterSurface, buildCampaignSystems(config).workflowRegistry())
	if err != nil {
		errs = append(errs, err)
	}
	sessions, err := campaignsessions.NewHandlerServices(newSessionSurfaceConfig(config))
	if err != nil {
		errs = append(errs, err)
	}
	invites, err := campaigninvites.NewHandlerServices(newInviteSurfaceConfig(config))
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
		Sessions:     sessions,
		Invites:      invites,
	}, nil
}

// newHandlers builds package wiring for this web seam from narrow app-facing
// contracts. Returns an error when core services are absent — this catches
// callers that skip the builder-level validation in newProductionHandlerServices.
func newHandlers(config handlersConfig) (handlers, error) {
	services := config.Services
	if services.Page.Workspace == nil || services.Catalog.campaigns == nil {
		return handlers{}, fmt.Errorf("campaigns module missing required services")
	}

	support := campaigndetail.NewSupport(config.Base, config.RequestMeta, config.Sync)
	detail := campaigndetail.NewHandler(support, services.Page)

	return handlers{
		catalog:      newCatalogHandlers(support, services.Catalog, config.Systems),
		starters:     newStarterHandlers(support, services.Starter),
		overview:     campaignoverview.NewHandler(detail, services.Overview),
		participants: campaignparticipants.NewHandler(detail, services.Participants),
		characters:   campaigncharacters.NewHandler(detail, services.Characters),
		sessions:     campaignsessions.NewHandler(detail, services.Sessions, config.PlayFallbackPort, config.PlayLaunchGrant),
		invites:      campaigninvites.NewHandler(detail, services.Invites),
	}, nil
}
