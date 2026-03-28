package invite

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	invitegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// Compose builds the invite module from the exact startup dependencies the area
// owns.
func Compose(
	inviteClient invitegateway.InviteClient,
	authClient invitegateway.AuthClient,
	requestMeta requestmeta.SchemePolicy,
	principal principal.PrincipalResolver,
	dashboardSync dashboardsync.Service,
) module.Module {
	gateway := invitegateway.NewGRPCGateway(inviteClient, authClient)
	return New(Config{
		Service:       inviteapp.NewService(gateway),
		RequestMeta:   requestMeta,
		Principal:     principal,
		DashboardSync: dashboardSync,
	})
}
