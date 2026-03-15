package campaigns

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// CampaignClient keeps composition ownership on the generated campaign client
// while this area composes narrower read and mutation seams internally.
type CampaignClient interface {
	campaigngateway.CampaignReadClient
	campaigngateway.CampaignMutationClient
}

// DiscoveryClient keeps composition ownership on the generated discovery client.
type DiscoveryClient interface {
	campaigngateway.DiscoveryClient
}

// ForkClient keeps composition ownership on the generated fork client.
type ForkClient interface {
	campaigngateway.ForkClient
}

// ParticipantClient keeps composition ownership on the generated participant
// client while this area composes narrower read and mutation seams internally.
type ParticipantClient interface {
	campaigngateway.ParticipantReadClient
	campaigngateway.ParticipantMutationClient
}

// CharacterClient keeps composition ownership on the generated character
// client while this area composes narrower read and mutation seams internally.
type CharacterClient interface {
	campaigngateway.CharacterReadClient
	campaigngateway.CharacterMutationClient
}

// SessionClient keeps composition ownership on the generated session client
// while this area composes narrower read and mutation seams internally.
type SessionClient interface {
	campaigngateway.SessionReadClient
	campaigngateway.SessionMutationClient
}

// InviteClient keeps composition ownership on the generated invite client
// while this area composes narrower read and mutation seams internally.
type InviteClient interface {
	campaigngateway.InviteReadClient
	campaigngateway.InviteMutationClient
}

// CompositionConfig owns the startup wiring required to construct the
// production campaigns module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	Base             modulehandler.Base
	ChatFallbackPort string
	DashboardSync    DashboardSync
	AssetBaseURL     string

	CampaignClient           CampaignClient
	InteractionClient        campaigngateway.InteractionClient
	DiscoveryClient          DiscoveryClient
	AgentClient              campaigngateway.AgentClient
	ParticipantClient        ParticipantClient
	CharacterClient          CharacterClient
	DaggerheartContentClient campaigngateway.DaggerheartContentClient
	DaggerheartAssetClient   campaigngateway.DaggerheartAssetClient
	SessionClient            SessionClient
	InviteClient             InviteClient
	SocialClient             campaigngateway.SocialClient
	AuthClient               campaigngateway.AuthClient
	AuthorizationClient      campaigngateway.AuthorizationClient
	ForkClient               ForkClient
}

// ProtectedSurfaceOptions carries the shared cross-cutting inputs the protected
// registry is allowed to pass into campaign composition.
type ProtectedSurfaceOptions struct {
	Base             modulehandler.Base
	ChatFallbackPort string
	DashboardSync    DashboardSync
	AssetBaseURL     string
}

// Compose builds the production campaigns module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	workflows := campaignworkflow.Registry{
		campaignapp.GameSystemDaggerheart: daggerheart.New(config.AssetBaseURL),
	}
	serviceConfig := newServiceConfigFromGRPCDeps(newGatewayDeps(config), config.AssetBaseURL)
	return New(Config{
		Services:         newHandlerServices(serviceConfig),
		Base:             config.Base,
		ChatFallbackPort: config.ChatFallbackPort,
		Workflows:        workflows,
		DashboardSync:    config.DashboardSync,
	})
}

// ComposeProtected composes the protected campaigns surface when the owning
// dependency set is complete. The registry only provides shared options and
// stable module ordering.
func ComposeProtected(options ProtectedSurfaceOptions, deps Dependencies) (module.Module, bool) {
	config := CompositionConfig{
		Base:                     options.Base,
		ChatFallbackPort:         options.ChatFallbackPort,
		DashboardSync:            options.DashboardSync,
		AssetBaseURL:             options.AssetBaseURL,
		CampaignClient:           deps.CampaignClient,
		InteractionClient:        deps.InteractionClient,
		DiscoveryClient:          deps.DiscoveryClient,
		AgentClient:              deps.AgentClient,
		ParticipantClient:        deps.ParticipantClient,
		CharacterClient:          deps.CharacterClient,
		DaggerheartContentClient: deps.DaggerheartContentClient,
		DaggerheartAssetClient:   deps.DaggerheartAssetClient,
		SessionClient:            deps.SessionClient,
		InviteClient:             deps.InviteClient,
		SocialClient:             deps.SocialClient,
		AuthClient:               deps.AuthClient,
		AuthorizationClient:      deps.AuthorizationClient,
		ForkClient:               deps.ForkClient,
	}
	if !config.configured() {
		return nil, false
	}
	return Compose(config), true
}

// configured reports whether the campaigns surface has the full dependency set
// required for production composition.
func (config CompositionConfig) configured() bool {
	return config.CampaignClient != nil &&
		config.InteractionClient != nil &&
		config.DiscoveryClient != nil &&
		config.AgentClient != nil &&
		config.ParticipantClient != nil &&
		config.CharacterClient != nil &&
		config.DaggerheartContentClient != nil &&
		config.DaggerheartAssetClient != nil &&
		config.SessionClient != nil &&
		config.InviteClient != nil &&
		config.SocialClient != nil &&
		config.AuthClient != nil &&
		config.AuthorizationClient != nil &&
		config.ForkClient != nil
}
