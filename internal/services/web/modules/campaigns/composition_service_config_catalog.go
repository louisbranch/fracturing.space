package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// newCatalogServiceConfig keeps catalog composition local to the catalog
// capability instead of routing it through one composition sink.
func newCatalogServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.CatalogServiceConfig {
	return campaignapp.CatalogServiceConfig{
		Read:     campaigngateway.NewCatalogReadGateway(deps.CatalogRead, assetBaseURL),
		Mutation: campaigngateway.NewCatalogMutationGateway(deps.CatalogMutation),
	}
}

// newWorkspaceServiceConfig keeps workspace-read composition with the catalog
// family instead of widening the root builder.
func newWorkspaceServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.WorkspaceServiceConfig {
	return campaignapp.WorkspaceServiceConfig{
		Read: campaigngateway.NewWorkspaceReadGateway(deps.WorkspaceRead, assetBaseURL),
	}
}

// newGameServiceConfig keeps game-surface composition isolated from generic
// workspace wiring.
func newGameServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.GameServiceConfig {
	return campaignapp.GameServiceConfig{
		Read: campaigngateway.NewGameReadGateway(deps.GameRead),
	}
}
