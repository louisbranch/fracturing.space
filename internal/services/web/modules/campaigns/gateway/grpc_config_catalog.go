package gateway

import campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"

// CatalogReadDeps keeps campaign catalog query dependencies explicit.
type CatalogReadDeps struct {
	Campaign CampaignReadClient
}

// StarterDeps keeps protected starter preview/launch dependencies explicit.
type StarterDeps struct {
	Discovery        DiscoveryClient
	Agent            AgentClient
	CampaignArtifact CampaignArtifactClient
	Campaign         CampaignMutationClient
	Fork             ForkClient
}

// CatalogMutationDeps keeps campaign catalog mutation dependencies explicit.
type CatalogMutationDeps struct {
	Campaign CampaignMutationClient
}

// WorkspaceReadDeps keeps workspace query dependencies explicit.
type WorkspaceReadDeps struct {
	Campaign CampaignReadClient
}

// catalogReadGateway maps campaign catalog reads from the campaign backend only.
type catalogReadGateway struct {
	read         CatalogReadDeps
	assetBaseURL string
}

// starterGateway maps protected starter preview and launch operations from
// discovery/game/AI deps.
type starterGateway struct {
	deps StarterDeps
}

// catalogMutationGateway maps campaign catalog mutations without widening read deps.
type catalogMutationGateway struct {
	mutation CatalogMutationDeps
}

// workspaceReadGateway maps workspace summary reads from campaign state.
type workspaceReadGateway struct {
	read         WorkspaceReadDeps
	assetBaseURL string
}

// NewCatalogReadGateway builds the campaign catalog read adapter from explicit
// dependencies.
func NewCatalogReadGateway(readDeps CatalogReadDeps, assetBaseURL string) campaignapp.CampaignCatalogReadGateway {
	if readDeps.Campaign == nil {
		return nil
	}
	return catalogReadGateway{
		read:         readDeps,
		assetBaseURL: assetBaseURL,
	}
}

// NewCatalogMutationGateway builds the campaign catalog mutation adapter from
// explicit dependencies.
func NewCatalogMutationGateway(mutationDeps CatalogMutationDeps) campaignapp.CampaignCatalogMutationGateway {
	if mutationDeps.Campaign == nil {
		return nil
	}
	return catalogMutationGateway{mutation: mutationDeps}
}

// NewWorkspaceReadGateway builds the workspace read adapter from explicit
// dependencies.
func NewWorkspaceReadGateway(readDeps WorkspaceReadDeps, assetBaseURL string) campaignapp.CampaignWorkspaceReadGateway {
	if readDeps.Campaign == nil {
		return nil
	}
	return workspaceReadGateway{
		read:         readDeps,
		assetBaseURL: assetBaseURL,
	}
}
