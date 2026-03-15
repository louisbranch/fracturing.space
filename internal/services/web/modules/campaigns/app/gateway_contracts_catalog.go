package app

import "context"

// CampaignCatalogReadGateway loads campaign list reads for the web service.
type CampaignCatalogReadGateway interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
}

// CampaignStarterGateway loads protected starter preview state and launches starter forks.
type CampaignStarterGateway interface {
	StarterPreview(context.Context, string) (CampaignStarterPreview, error)
	LaunchStarter(context.Context, string, LaunchStarterInput) (StarterLaunchResult, error)
}

// CampaignWorkspaceReadGateway loads campaign workspace metadata reads for the web service.
type CampaignWorkspaceReadGateway interface {
	CampaignName(context.Context, string) (string, error)
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
}

// CampaignCatalogMutationGateway applies campaign catalog mutations for the web service.
type CampaignCatalogMutationGateway interface {
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
}

// CampaignConfigurationMutationGateway applies campaign-level settings mutations for the web service.
type CampaignConfigurationMutationGateway interface {
	UpdateCampaign(context.Context, string, UpdateCampaignInput) error
}
