package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// newPageServiceConfig keeps shared detail-page wiring close to the catalog and
// workspace ownership it depends on.
func newPageServiceConfig(config CompositionConfig) pageServiceConfig {
	return pageServiceConfig{
		Workspace:     newWorkspaceServiceConfig(config),
		SessionRead:   newSessionReadServiceConfig(config),
		Authorization: newPageAuthorizationGateway(config),
	}
}

// newPageAuthorizationGateway keeps shared detail-page authorization wiring
// close to the workspace shell that consumes it.
func newPageAuthorizationGateway(config CompositionConfig) campaignapp.AuthorizationGateway {
	return campaigngateway.NewAuthorizationGateway(config.Gateway.Page.Authorization)
}

// newCatalogSurfaceConfig keeps catalog composition local to the catalog route
// surface.
func newCatalogSurfaceConfig(config CompositionConfig) catalogServiceConfig {
	return catalogServiceConfig{
		Catalog: newCatalogServiceConfig(config),
	}
}

// newCatalogServiceConfig keeps catalog composition local to the catalog
// capability instead of routing it through one composition sink.
func newCatalogServiceConfig(config CompositionConfig) campaignapp.CatalogServiceConfig {
	return campaignapp.CatalogServiceConfig{
		Read:     campaigngateway.NewCatalogReadGateway(config.Gateway.Catalog.Read, config.Options.AssetBaseURL),
		Mutation: campaigngateway.NewCatalogMutationGateway(config.Gateway.Catalog.Mutation),
	}
}

// newStarterSurfaceConfig keeps protected starter composition local to the
// starter surface.
func newStarterSurfaceConfig(config CompositionConfig) starterServiceConfig {
	return starterServiceConfig{
		Starter: newStarterServiceConfig(config),
	}
}

// newStarterServiceConfig keeps protected starter preview/launch composition local to starter ownership.
func newStarterServiceConfig(config CompositionConfig) campaignapp.StarterServiceConfig {
	return campaignapp.StarterServiceConfig{
		Gateway: campaigngateway.NewStarterGateway(config.Gateway.Starter.Starter),
	}
}

// newWorkspaceServiceConfig keeps workspace-read composition with the catalog
// family instead of widening the root builder.
func newWorkspaceServiceConfig(config CompositionConfig) campaignapp.WorkspaceServiceConfig {
	return campaignapp.WorkspaceServiceConfig{
		Read: campaigngateway.NewWorkspaceReadGateway(config.Gateway.Page.Workspace, config.Options.AssetBaseURL),
	}
}

// newPageGatewayDeps groups the shared page-detail gateway clients needed by
// multiple campaign detail surfaces.
func newPageGatewayDeps(deps Dependencies) campaigngateway.PageGatewayDeps {
	return campaigngateway.PageGatewayDeps{
		Workspace:     campaigngateway.WorkspaceReadDeps{Campaign: deps.CampaignClient},
		SessionRead:   campaigngateway.SessionReadDeps{Session: deps.SessionClient, Campaign: deps.CampaignClient},
		Authorization: campaigngateway.AuthorizationDeps{Client: deps.AuthorizationClient},
	}
}

// newCatalogGatewayDeps keeps catalog gateway grouping aligned with the catalog
// route surface instead of one module-wide dependency bag.
func newCatalogGatewayDeps(deps Dependencies) campaigngateway.CatalogGatewayDeps {
	return campaigngateway.CatalogGatewayDeps{
		Read:     campaigngateway.CatalogReadDeps{Campaign: deps.CampaignClient},
		Mutation: campaigngateway.CatalogMutationDeps{Campaign: deps.CampaignClient},
	}
}

// newStarterGatewayDeps keeps starter launch dependencies local to the starter
// surface so new system support stays area-owned.
func newStarterGatewayDeps(deps Dependencies) campaigngateway.StarterGatewayDeps {
	return campaigngateway.StarterGatewayDeps{
		Starter: campaigngateway.StarterDeps{
			Discovery:        deps.DiscoveryClient,
			Agent:            deps.AgentClient,
			CampaignArtifact: deps.CampaignArtifactClient,
			Campaign:         deps.CampaignClient,
			Fork:             deps.ForkClient,
		},
	}
}
