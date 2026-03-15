package campaigns

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated campaign workspace routes.
type Module struct {
	services         handlerServices
	base             modulehandler.Base
	playFallbackPort string
	playLaunchGrant  playlaunchgrant.Config
	requestMeta      requestmeta.SchemePolicy
	workflows        campaignworkflow.Registry
	sync             DashboardSync
	mountErr         error
}

// Config defines constructor dependencies for a campaigns module.
type Config struct {
	Services         handlerServices
	Base             modulehandler.Base
	PlayFallbackPort string
	PlayLaunchGrant  playlaunchgrant.Config
	RequestMeta      requestmeta.SchemePolicy
	Workflows        campaignworkflow.Registry
	DashboardSync    DashboardSync
}

// New returns a campaigns module with explicit dependencies.
func New(config Config) Module {
	return Module{
		services:         config.Services,
		base:             config.Base,
		playFallbackPort: config.PlayFallbackPort,
		playLaunchGrant:  config.PlayLaunchGrant,
		requestMeta:      config.RequestMeta,
		workflows:        config.Workflows,
		sync:             config.DashboardSync,
		mountErr:         validateHandlerServices(config.Services),
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "campaigns" }

// Healthy reports whether the campaigns module has an operational gateway.
func (m Module) Healthy() bool {
	return m.mountErr == nil
}

// Mount wires campaign route handlers.
func (m Module) Mount() (module.Mount, error) {
	if m.mountErr != nil {
		return module.Mount{}, m.mountErr
	}
	mux := http.NewServeMux()
	h := newHandlers(handlersConfig{
		Services:         m.services,
		Base:             m.base,
		PlayFallbackPort: m.playFallbackPort,
		PlayLaunchGrant:  m.playLaunchGrant,
		RequestMeta:      m.requestMeta,
		Sync:             m.sync,
		Workflows:        m.workflows,
	})
	registerStableRoutes(mux, h)
	return module.Mount{Prefix: routepath.CampaignsPrefix, CanonicalRoot: true, Handler: mux}, nil
}

// validateHandlerServices enforces fail-fast route wiring once campaigns is selected for mounting.
func validateHandlerServices(services handlerServices) error {
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
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("campaigns module missing required services: %s", strings.Join(missing, ", "))
}
