package app

// CatalogServiceConfig keeps campaign catalog dependencies explicit.
type CatalogServiceConfig struct {
	Read     CampaignCatalogReadGateway
	Mutation CampaignCatalogMutationGateway
}

// StarterServiceConfig keeps protected starter preview/launch dependencies explicit.
type StarterServiceConfig struct {
	Gateway CampaignStarterGateway
}

// WorkspaceServiceConfig keeps workspace-read dependencies explicit.
type WorkspaceServiceConfig struct {
	Read CampaignWorkspaceReadGateway
}
