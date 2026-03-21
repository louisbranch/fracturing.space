package modules

import (
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/invite"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	"google.golang.org/grpc"
)

// NewDependencies returns module dependency defaults with shared runtime
// configuration applied.
func NewDependencies(assetBaseURL string) Dependencies {
	return Dependencies{AssetBaseURL: assetBaseURL}
}

// BindAuthDependency wires auth-backed clients into the module dependency set.
func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	publicauth.BindAuthDependency(&deps.PublicAuth, conn)
	profile.BindAuthDependency(&deps.Profile, conn)
	settings.BindAuthDependency(&deps.Settings, conn)
	campaigns.BindAuthDependency(&deps.Campaigns, conn)
	invite.BindAuthDependency(&deps.Invite, conn)
}

// BindSocialDependency wires social-backed clients into the module dependency
// set.
func BindSocialDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	profile.BindSocialDependency(&deps.Profile, conn)
	settings.BindSocialDependency(&deps.Settings, conn)
	campaigns.BindSocialDependency(&deps.Campaigns, conn)
}

// BindGameDependency wires game-backed clients into the module dependency set.
func BindGameDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	campaigns.BindGameDependency(&deps.Campaigns, conn)
	deps.DashboardSync.GameEventClient = statev1.NewEventServiceClient(conn)
}

// BindInviteDependency wires invite-service clients into the module dependency set.
func BindInviteDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	campaigns.BindInviteDependency(&deps.Campaigns, conn)
	invite.BindInviteDependency(&deps.Invite, conn)
}

// BindAIDependency wires AI-backed clients into the module dependency set.
func BindAIDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	settings.BindAIDependency(&deps.Settings, conn)
	campaigns.BindAIDependency(&deps.Campaigns, conn)
}

// BindDiscoveryDependency wires discovery-backed clients into the module
// dependency set.
func BindDiscoveryDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	discovery.BindDependency(&deps.Discovery, conn)
	campaigns.BindDiscoveryDependency(&deps.Campaigns, conn)
}

// BindUserHubDependency wires userhub-backed clients into the module
// dependency set.
func BindUserHubDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	dashboard.BindUserHubDependency(&deps.Dashboard, conn)
	deps.DashboardSync.UserHubControlClient = userhubv1.NewUserHubControlServiceClient(conn)
}

// BindNotificationsDependency wires notification-backed clients into the
// module dependency set.
func BindNotificationsDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	notifications.BindDependency(&deps.Notifications, conn)
}

// BindStatusDependency wires the status client into the dashboard dependency
// set.
func BindStatusDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	dashboard.BindStatusDependency(&deps.Dashboard, conn)
}
