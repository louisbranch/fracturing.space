package web

import (
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/invite"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	grpc "google.golang.org/grpc"
)

// DependencyBundle is a single source of startup dependencies used by web service
// composition.
type DependencyBundle struct {
	// Principal carries the clients required for request-scoped principal resolution.
	Principal principal.Dependencies
	// Modules carries feature module dependencies and shared runtime config.
	Modules modules.Dependencies
}

// NewDependencyBundle returns a dependency bundle with shared runtime config
// pre-applied to both principal and module dependency sets.
func NewDependencyBundle(assetBaseURL string) DependencyBundle {
	return DependencyBundle{
		Principal: principal.NewDependencies(assetBaseURL),
		Modules:   modules.NewDependencies(assetBaseURL),
	}
}

// BindAuthDependency wires auth-backed clients into the web dependency bundle.
func BindAuthDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	principal.BindAuthDependency(&bundle.Principal, conn)
	publicauth.BindAuthDependency(&bundle.Modules.PublicAuth, conn)
	profile.BindAuthDependency(&bundle.Modules.Profile, conn)
	settings.BindAuthDependency(&bundle.Modules.Settings, conn)
	campaigns.BindAuthDependency(&bundle.Modules.Campaigns, conn)
	invite.BindAuthDependency(&bundle.Modules.Invite, conn)
}

// BindSocialDependency wires social-backed clients into the web dependency bundle.
func BindSocialDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	principal.BindSocialDependency(&bundle.Principal, conn)
	profile.BindSocialDependency(&bundle.Modules.Profile, conn)
	settings.BindSocialDependency(&bundle.Modules.Settings, conn)
	campaigns.BindSocialDependency(&bundle.Modules.Campaigns, conn)
}

// BindGameDependency wires game-backed clients into the web dependency bundle.
func BindGameDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	campaigns.BindGameDependency(&bundle.Modules.Campaigns, conn)
	bundle.Modules.DashboardSync.GameEventClient = statev1.NewEventServiceClient(conn)
}

// BindInviteDependency wires invite-service clients into the web dependency bundle.
func BindInviteDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	campaigns.BindInviteDependency(&bundle.Modules.Campaigns, conn)
	invite.BindInviteDependency(&bundle.Modules.Invite, conn)
}

// BindAIDependency wires AI-backed clients into the web dependency bundle.
func BindAIDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	settings.BindAIDependency(&bundle.Modules.Settings, conn)
	campaigns.BindAIDependency(&bundle.Modules.Campaigns, conn)
}

// BindDiscoveryDependency wires discovery-backed clients into the web dependency bundle.
func BindDiscoveryDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	discovery.BindDependency(&bundle.Modules.Discovery, conn)
	campaigns.BindDiscoveryDependency(&bundle.Modules.Campaigns, conn)
}

// BindUserHubDependency wires userhub-backed clients into the web dependency bundle.
func BindUserHubDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	dashboard.BindUserHubDependency(&bundle.Modules.Dashboard, conn)
	bundle.Modules.DashboardSync.UserHubControlClient = userhubv1.NewUserHubControlServiceClient(conn)
}

// BindNotificationsDependency wires notification-backed clients into the web dependency bundle.
func BindNotificationsDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	principal.BindNotificationsDependency(&bundle.Principal, conn)
	notifications.BindDependency(&bundle.Modules.Notifications, conn)
}

// BindStatusDependency wires the status client into the dashboard dependency set.
func BindStatusDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	dashboard.BindStatusDependency(&bundle.Modules.Dashboard, conn)
}
