package app

import "context"

// CampaignCatalogService exposes campaign catalog reads and mutations.
type CampaignCatalogService interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
}

// CampaignStarterService exposes protected starter preview and launch flows.
type CampaignStarterService interface {
	StarterPreview(context.Context, string) (CampaignStarterPreview, error)
	LaunchStarter(context.Context, string, LaunchStarterInput) (StarterLaunchResult, error)
}

// CampaignWorkspaceService exposes campaign workspace reads used by transport.
type CampaignWorkspaceService interface {
	CampaignName(context.Context, string) string
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
}

// CampaignConfigurationService exposes campaign-level settings mutations.
type CampaignConfigurationService interface {
	UpdateCampaign(context.Context, string, UpdateCampaignInput) error
}

// CampaignAuthorizationService exposes transport-facing authorization checks.
type CampaignAuthorizationService interface {
	RequireManageCampaign(context.Context, string) error
	RequireManageSession(context.Context, string) error
	RequireManageParticipants(context.Context, string) error
	RequireManageInvites(context.Context, string) error
	RequireMutateCharacters(context.Context, string) error
}
