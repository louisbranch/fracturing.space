package campaigns

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated campaign workspace routes.
type Module struct {
	services         handlerServices
	base             modulehandler.Base
	chatFallbackPort string
	workflows        campaignworkflow.Registry
	sync             DashboardSync
	mountErr         error
}

// Config defines constructor dependencies for a campaigns module.
type Config struct {
	Services         handlerServices
	Base             modulehandler.Base
	ChatFallbackPort string
	Workflows        campaignworkflow.Registry
	DashboardSync    DashboardSync
}

// New returns a campaigns module with explicit dependencies.
func New(config Config) Module {
	return Module{
		services:         config.Services,
		base:             config.Base,
		chatFallbackPort: config.ChatFallbackPort,
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
		ChatFallbackPort: m.chatFallbackPort,
		Sync:             m.sync,
		Workflows:        m.workflows,
	})
	registerStableRoutes(mux, h)
	return module.Mount{Prefix: routepath.CampaignsPrefix, Handler: mux}, nil
}

// validateHandlerServices enforces fail-fast route wiring once campaigns is selected for mounting.
func validateHandlerServices(services handlerServices) error {
	missing := []string{}
	if services.Catalog == nil {
		missing = append(missing, "catalog")
	}
	if services.Workspace == nil {
		missing = append(missing, "workspace")
	}
	if services.Game == nil {
		missing = append(missing, "game")
	}
	if services.ParticipantReads == nil {
		missing = append(missing, "participant-reads")
	}
	if services.ParticipantMutate == nil {
		missing = append(missing, "participant-mutation")
	}
	if services.AutomationReads == nil {
		missing = append(missing, "automation-reads")
	}
	if services.AutomationMutation == nil {
		missing = append(missing, "automation-mutation")
	}
	if services.CharacterReads == nil {
		missing = append(missing, "character-reads")
	}
	if services.CharacterControl == nil {
		missing = append(missing, "character-control")
	}
	if services.CharacterMutation == nil {
		missing = append(missing, "character-mutation")
	}
	if services.SessionReads == nil {
		missing = append(missing, "session-reads")
	}
	if services.SessionMutation == nil {
		missing = append(missing, "session-mutation")
	}
	if services.InviteReads == nil {
		missing = append(missing, "invite-reads")
	}
	if services.InviteMutation == nil {
		missing = append(missing, "invite-mutation")
	}
	if services.Configuration == nil {
		missing = append(missing, "configuration")
	}
	if services.Authorization == nil {
		missing = append(missing, "authorization")
	}
	if services.CreationPages == nil {
		missing = append(missing, "creation-pages")
	}
	if services.CreationFlow == nil {
		missing = append(missing, "creation-flow")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("campaigns module missing required services: %s", strings.Join(missing, ", "))
}
