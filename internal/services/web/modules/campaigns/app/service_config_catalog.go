package app

// CatalogServiceConfig keeps campaign catalog dependencies explicit.
type CatalogServiceConfig struct {
	Read     CampaignCatalogReadGateway
	Mutation CampaignCatalogMutationGateway
}

// WorkspaceServiceConfig keeps workspace-read dependencies explicit.
type WorkspaceServiceConfig struct {
	Read CampaignWorkspaceReadGateway
}

// GameServiceConfig keeps game-surface dependencies explicit.
type GameServiceConfig struct {
	Read CampaignGameReadGateway
}
