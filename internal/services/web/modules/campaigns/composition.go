package campaigns

import (
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
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
	Options ProtectedSurfaceOptions
	Gateway campaigngateway.GRPCGatewayDeps
}

// ProtectedSurfaceOptions carries the shared cross-cutting inputs the protected
// registry is allowed to pass into campaign composition.
type ProtectedSurfaceOptions struct {
	Base             modulehandler.Base
	PlayFallbackPort string
	PlayLaunchGrant  playlaunchgrant.Config
	RequestMeta      requestmeta.SchemePolicy
	DashboardSync    DashboardSync
	AssetBaseURL     string
}

// Compose builds the production campaigns module from area-owned startup
// dependencies. Dependency validation errors are stored in the module and
// surface when Mount is called.
func Compose(config CompositionConfig) module.Module {
	services, err := newProductionHandlerServices(config)
	if err != nil {
		return Module{mountErr: err}
	}
	return New(Config{
		Services:         services,
		Base:             config.Options.Base,
		PlayFallbackPort: config.Options.PlayFallbackPort,
		PlayLaunchGrant:  config.Options.PlayLaunchGrant,
		RequestMeta:      config.Options.RequestMeta,
		Systems:          buildCampaignSystems(config),
		DashboardSync:    config.Options.DashboardSync,
	})
}

// ComposeProtected composes the protected campaigns surface when the owning
// dependency set is complete. The registry only provides shared options and
// stable module ordering.
func ComposeProtected(options ProtectedSurfaceOptions, deps Dependencies) (module.Module, bool) {
	if !deps.configured() {
		return nil, false
	}
	return Compose(newCompositionConfig(options, deps)), true
}

// newCompositionConfig groups route-surface gateway deps before module
// construction so the registry does not rebuild campaign startup wiring inline.
func newCompositionConfig(options ProtectedSurfaceOptions, deps Dependencies) CompositionConfig {
	return CompositionConfig{
		Options: options,
		Gateway: campaigngateway.GRPCGatewayDeps{
			Page:         newPageGatewayDeps(deps),
			Catalog:      newCatalogGatewayDeps(deps),
			Starter:      newStarterGatewayDeps(deps),
			Overview:     newOverviewGatewayDeps(deps),
			Participants: newParticipantGatewayDeps(deps),
			Characters:   newCharacterGatewayDeps(deps),
			Sessions:     newSessionGatewayDeps(deps),
			Invites:      newInviteGatewayDeps(deps),
		},
	}
}

// configured reports whether the campaigns dependency set has the full client
// graph required for production composition.
func (deps Dependencies) configured() bool {
	return deps.CampaignClient != nil &&
		deps.DiscoveryClient != nil &&
		deps.AgentClient != nil &&
		deps.CampaignArtifactClient != nil &&
		deps.ParticipantClient != nil &&
		deps.CharacterClient != nil &&
		deps.DaggerheartContentClient != nil &&
		deps.DaggerheartAssetClient != nil &&
		deps.SessionClient != nil &&
		deps.InviteClient != nil &&
		deps.SocialClient != nil &&
		deps.AuthClient != nil &&
		deps.AuthorizationClient != nil &&
		deps.ForkClient != nil
}
