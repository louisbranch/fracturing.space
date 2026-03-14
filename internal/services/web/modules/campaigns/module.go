package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated campaign workspace routes.
type Module struct {
	gateway          campaignapp.CampaignGateway
	base             modulehandler.Base
	chatFallbackPort string
	workflows        campaignworkflow.Registry
	sync             DashboardSync
}

// Config defines constructor dependencies for a campaigns module.
type Config struct {
	Gateway          campaignapp.CampaignGateway
	Base             modulehandler.Base
	ChatFallbackPort string
	Workflows        campaignworkflow.Registry
	DashboardSync    DashboardSync
}

// New returns a campaigns module with explicit dependencies.
func New(config Config) Module {
	return Module{
		gateway:          config.Gateway,
		base:             config.Base,
		chatFallbackPort: config.ChatFallbackPort,
		workflows:        config.Workflows,
		sync:             config.DashboardSync,
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
	svc := campaignapp.NewService(m.gateway)
	h := newHandlers(svc, m.base, m.chatFallbackPort, m.sync, m.workflows)
	registerStableRoutes(mux, h)
	return module.Mount{Prefix: routepath.CampaignsPrefix, Handler: mux}, nil
}
