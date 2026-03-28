package invite

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	invitegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// CompositionConfig owns the startup wiring required to construct the
// production invite module.
type CompositionConfig struct {
	InviteClient  invitegateway.InviteClient
	AuthClient    invitegateway.AuthClient
	RequestMeta   requestmeta.SchemePolicy
	Principal     principal.PrincipalResolver
	DashboardSync dashboardsync.Service
}

// Compose builds the invite module from the exact startup dependencies the area
// owns.
func Compose(config CompositionConfig) module.Module {
	gateway := invitegateway.NewGRPCGateway(config.InviteClient, config.AuthClient)
	return New(Config{
		Service:       inviteapp.NewService(gateway),
		RequestMeta:   config.RequestMeta,
		Principal:     config.Principal,
		DashboardSync: config.DashboardSync,
	})
}
