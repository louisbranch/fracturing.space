// Package modules defines web module registry helpers.
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
// Request-scoped principal resolution is provided separately via
// platform/requestresolver since it is derived by the server after
// construction.
type Dependencies struct {
	AssetBaseURL string

	// Campaigns owns all campaign/session/character/invite/authz clients.
	Campaigns CampaignDependencies

	// Invite owns public invite reads/mutations and invite-driven dashboard sync.
	Invite InviteDependencies

	// Dashboard owns userhub and status health dependencies.
	Dashboard DashboardDependencies

	// Profile owns public profile lookup dependencies.
	Profile ProfileDependencies

	// Settings owns profile/account/credential dependencies.
	Settings SettingsDependencies

	// DashboardSync owns cross-module dashboard freshness dependencies.
	DashboardSync DashboardSyncDependencies

	// PublicAuth owns authentication/session dependencies.
	PublicAuth PublicAuthDependencies

	// Notifications owns inbox notification dependencies.
	Notifications NotificationDependencies

	// Discovery owns starter/discovery list dependencies.
	Discovery DiscoveryDependencies
}

// CampaignDependencies contains campaign feature clients.
type CampaignDependencies = campaigns.Dependencies

// CampaignClient keeps composition ownership on the generated campaigns client
// while the gateway consumes narrower read/mutation bundles internally.
type CampaignClient = campaigns.CampaignClient

// ParticipantClient keeps one composition-owned participant client while the
// gateway consumes separate read and mutation seams internally.
type ParticipantClient = campaigns.ParticipantClient

// CharacterClient keeps one composition-owned character client while the
// gateway consumes separate read and mutation seams internally.
type CharacterClient = campaigns.CharacterClient

// SessionClient keeps one composition-owned session client while the gateway
// consumes separate read and mutation seams internally.
type SessionClient = campaigns.SessionClient

// InviteClient keeps one composition-owned invite client while the gateway
// consumes separate read and mutation seams internally.
type InviteClient = campaigns.InviteClient

// InviteDependencies contains public-invite feature clients.
type InviteDependencies = invite.Dependencies

// DashboardDependencies contains dashboard feature clients.
type DashboardDependencies = dashboard.Dependencies

// ProfileDependencies contains profile feature clients.
type ProfileDependencies = profile.Dependencies

// SettingsDependencies contains settings feature clients.
type SettingsDependencies = settings.Dependencies

// DashboardSyncDependencies contains shared mutation-sync clients.
type DashboardSyncDependencies struct {
	UserHubControlClient dashboardsync.UserHubControlClient
	GameEventClient      dashboardsync.GameEventClient
}

// PublicAuthDependencies contains public-auth feature clients.
type PublicAuthDependencies = publicauth.Dependencies

// NotificationDependencies contains notification feature clients.
type NotificationDependencies = notifications.Dependencies

// DiscoveryDependencies contains discovery feature clients.
type DiscoveryDependencies = discovery.Dependencies
