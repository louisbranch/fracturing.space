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
}

// New returns a campaigns module with zero-value dependencies (degraded mode).
func New() Module {
	return Module{}
}

// NewStableWithGateway returns a campaigns module with stable route exposure.
func NewStableWithGateway(gateway CampaignGateway, base modulehandler.Base, chatFallbackPort string, workflows map[string]CharacterCreationWorkflow) Module {
	return Module{
		gateway:          gateway,
		base:             base,
		chatFallbackPort: chatFallbackPort,
		workflows:        workflows,
	}
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
	h := newHandlers(svc, m.base, m.chatFallbackPort, m.workflows)
	registerStableRoutes(mux, h)
	return module.Mount{Prefix: routepath.CampaignsPrefix, Handler: mux}, nil
}
