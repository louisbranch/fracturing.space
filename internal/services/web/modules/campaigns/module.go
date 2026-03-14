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
	readGateway      campaignapp.ReadGateway
	mutationGateway  campaignapp.MutationGateway
	authzGateway     campaignapp.AuthzGateway
	base             modulehandler.Base
	chatFallbackPort string
	workflows        campaignworkflow.Registry
	sync             DashboardSync
}

// Config defines constructor dependencies for a campaigns module.
type Config struct {
	ReadGateway      campaignapp.ReadGateway
	MutationGateway  campaignapp.MutationGateway
	AuthzGateway     campaignapp.AuthzGateway
	Base             modulehandler.Base
	ChatFallbackPort string
	Workflows        campaignworkflow.Registry
	DashboardSync    DashboardSync
}

// New returns a campaigns module with explicit dependencies.
func New(config Config) Module {
	return Module{
		readGateway:      config.ReadGateway,
		mutationGateway:  config.MutationGateway,
		authzGateway:     config.AuthzGateway,
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
	return campaignapp.IsGatewayHealthy(m.readGateway)
}

// Mount wires campaign route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := campaignapp.NewService(campaignapp.ServiceConfig{
		ReadGateway:     m.readGateway,
		MutationGateway: m.mutationGateway,
		AuthzGateway:    m.authzGateway,
	})
	h := newHandlers(svc, m.base, m.chatFallbackPort, m.sync, m.workflows)
	registerStableRoutes(mux, h)
	return module.Mount{Prefix: routepath.CampaignsPrefix, Handler: mux}, nil
}
