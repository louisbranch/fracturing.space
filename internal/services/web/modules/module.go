package modules

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/invite"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
)

// Dependencies carries the gRPC clients and shared config required to compose
// the web module registry. Each client field is typed as the narrow interface
// defined by the consuming module, so modules physically cannot access clients
// they were not given.
//
// Request-scoped principal resolution is provided separately via the
// principal package since it is derived by the server after construction.
type Dependencies struct {
	AssetBaseURL string

	// Campaigns owns all campaign/session/character/invite/authz clients.
	Campaigns campaigns.Dependencies

	// Invite owns public invite reads/mutations and invite-driven dashboard sync.
	Invite invite.Dependencies

	// Dashboard owns userhub and status health dependencies.
	Dashboard dashboard.Dependencies

	// Profile owns public profile lookup dependencies.
	Profile profile.Dependencies

	// Settings owns profile/account/credential dependencies.
	Settings settings.Dependencies

	// DashboardSync owns cross-module dashboard freshness dependencies.
	DashboardSync DashboardSyncDependencies

	// PublicAuth owns authentication/session dependencies.
	PublicAuth publicauth.Dependencies

	// Notifications owns inbox notification dependencies.
	Notifications notifications.Dependencies

	// Discovery owns starter/discovery list dependencies.
	Discovery discovery.Dependencies
}

// DashboardSyncDependencies contains shared mutation-sync clients.
type DashboardSyncDependencies struct {
	UserHubControlClient dashboardsync.UserHubControlClient
	GameEventClient      dashboardsync.GameEventClient
}
