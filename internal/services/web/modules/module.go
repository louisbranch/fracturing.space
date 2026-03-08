// Package modules defines web module registry helpers.
package modules

import (
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
)

// Mount aliases the module mount contract.
type Mount = module.Mount

// Module aliases the module interface contract.
type Module = module.Module

// ModuleResolvers carries request-scoped resolver functions derived from the
// principal resolver. The server constructs these after building the principal
// resolver and passes them to registry functions for module composition.
type ModuleResolvers struct {
	ResolveViewer   module.ResolveViewer
	ResolveSignedIn module.ResolveSignedIn
	ResolveUserID   module.ResolveUserID
	ResolveLanguage module.ResolveLanguage
}

// Dependencies carries the gRPC clients and shared config required to compose
// the web module registry. Each client field is typed as the narrow interface
// defined by the consuming module, so modules physically cannot access clients
// they were not given.
//
// Request-scoped resolvers (viewer, user-id, language) are provided separately
// via ModuleResolvers since they are derived by the server after construction.
type Dependencies struct {
	AssetBaseURL string

	// Campaigns owns all campaign/session/character/invite/authz clients.
	Campaigns CampaignDependencies

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
type CampaignDependencies struct {
	CampaignClient           campaigns.CampaignClient
	AgentClient              campaigns.AgentClient
	ParticipantClient        campaigns.ParticipantClient
	CharacterClient          campaigns.CharacterClient
	DaggerheartContentClient campaigns.DaggerheartContentClient
	DaggerheartAssetClient   campaigns.DaggerheartAssetClient
	SessionClient            campaigns.SessionClient
	InviteClient             campaigns.InviteClient
	AuthorizationClient      campaigns.AuthorizationClient
}

// DashboardDependencies contains dashboard feature clients.
type DashboardDependencies struct {
	UserHubClient dashboard.UserHubClient
	StatusClient  statusv1.StatusServiceClient
}

// ProfileDependencies contains profile feature clients.
type ProfileDependencies struct {
	SocialClient profile.SocialClient
}

// SettingsDependencies contains settings feature clients.
type SettingsDependencies struct {
	SocialClient     settings.SocialClient
	AccountClient    settings.AccountClient
	CredentialClient settings.CredentialClient
}

// DashboardSyncDependencies contains shared mutation-sync clients.
type DashboardSyncDependencies struct {
	UserHubControlClient dashboardsync.UserHubControlClient
	GameEventClient      dashboardsync.GameEventClient
}

// PublicAuthDependencies contains public-auth feature clients.
type PublicAuthDependencies struct {
	AuthClient publicauthgateway.AuthClient
}

// NotificationDependencies contains notification feature clients.
type NotificationDependencies struct {
	NotificationClient notifications.NotificationClient
}

// DiscoveryDependencies contains discovery feature clients.
type DiscoveryDependencies struct {
	DiscoveryClient discovery.DiscoveryClient
}
