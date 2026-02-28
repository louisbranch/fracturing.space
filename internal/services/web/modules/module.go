// Package modules defines web module registry helpers.
package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
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

	// Campaign module clients.
	CampaignClient           campaigns.CampaignClient
	ParticipantClient        campaigns.ParticipantClient
	CharacterClient          campaigns.CharacterClient
	DaggerheartContentClient campaigns.DaggerheartContentClient
	SessionClient            campaigns.SessionClient
	InviteClient             campaigns.InviteClient
	AuthorizationClient      campaigns.AuthorizationClient

	// Dashboard module client.
	UserHubClient dashboard.UserHubClient

	// Settings module clients.
	SocialClient     settings.SocialClient
	AccountClient    settings.AccountClient
	CredentialClient settings.CredentialClient

	// Public auth module client.
	AuthClient publicauth.AuthClient

	// Notification module client.
	NotificationClient notifications.NotificationClient
}
