package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated campaign workspace routes.
type Module struct {
	gateway          CampaignGateway
	base             modulehandler.Base
	chatFallbackPort string
	workflows        map[string]CharacterCreationWorkflow
	registerRoutes   func(*http.ServeMux, handlers)
}

// New returns a campaigns module with zero-value dependencies (degraded mode).
func New() Module {
	return Module{registerRoutes: registerStableRoutes}
}

// NewStableWithGateway returns a campaigns module with stable route exposure.
func NewStableWithGateway(gateway CampaignGateway, base modulehandler.Base, chatFallbackPort string, workflows map[string]CharacterCreationWorkflow) Module {
	return Module{gateway: gateway, base: base, chatFallbackPort: chatFallbackPort, workflows: workflows, registerRoutes: registerStableRoutes}
}

// NewExperimentalWithGateway returns a campaigns module with experimental route exposure.
func NewExperimentalWithGateway(gateway CampaignGateway, base modulehandler.Base, chatFallbackPort string, workflows map[string]CharacterCreationWorkflow) Module {
	return Module{gateway: gateway, base: base, chatFallbackPort: chatFallbackPort, workflows: workflows, registerRoutes: registerExperimentalRoutes}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "campaigns" }

// Healthy reports whether the campaigns module has an operational gateway.
func (m Module) Healthy() bool {
	return campaignapp.IsGatewayHealthy(m.gateway)
}

// Mount wires campaign route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := newServiceWithWorkflows(m.gateway, m.workflows)
	h := newHandlers(svc, m.base, m.chatFallbackPort)
	if m.registerRoutes == nil {
		m.registerRoutes = registerStableRoutes
	}
	m.registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.CampaignsPrefix, Handler: mux}, nil
}
