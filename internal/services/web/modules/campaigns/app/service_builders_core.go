package app

import "errors"

// catalogService keeps catalog reads and mutations local to catalog ownership.
type catalogService struct {
	read     CampaignCatalogReadGateway
	mutation CampaignCatalogMutationGateway
}

// starterService keeps protected starter preview and launch behavior local to starter ownership.
type starterService struct {
	gateway CampaignStarterGateway
}

// workspaceService keeps campaign workspace summaries on their own read seam.
type workspaceService struct {
	read CampaignWorkspaceReadGateway
}

// authorizationSupport centralizes unary authorization checks shared by mutation services.
type authorizationSupport struct {
	gateway AuthorizationGateway
}

// authorizationService exposes transport-facing unary authorization checks.
type authorizationService struct {
	auth authorizationSupport
}

// NewCatalogService constructs the catalog-only service surface from explicit
// gateway seams. Returns an error when required dependencies are absent.
func NewCatalogService(config CatalogServiceConfig) (CampaignCatalogService, error) {
	if config.Read == nil || config.Mutation == nil {
		return nil, errors.New("catalog service: missing required gateway dependencies")
	}
	return catalogService{read: config.Read, mutation: config.Mutation}, nil
}

// NewStarterService constructs the protected starter service surface from
// explicit gateway seams. Returns an error when the gateway is absent.
func NewStarterService(config StarterServiceConfig) (CampaignStarterService, error) {
	if config.Gateway == nil {
		return nil, errors.New("starter service: missing required gateway dependency")
	}
	return starterService{gateway: config.Gateway}, nil
}

// NewWorkspaceService constructs the workspace-read service surface from
// explicit gateway seams. Returns an error when the read gateway is absent.
func NewWorkspaceService(config WorkspaceServiceConfig) (CampaignWorkspaceService, error) {
	if config.Read == nil {
		return nil, errors.New("workspace service: missing required read gateway")
	}
	return workspaceService{read: config.Read}, nil
}

// NewAuthorizationService constructs the authorization service surface from
// explicit gateway seams. Returns an error when the gateway is absent.
func NewAuthorizationService(authorization AuthorizationGateway) (CampaignAuthorizationService, error) {
	if authorization == nil {
		return nil, errors.New("authorization service: missing required gateway dependency")
	}
	return authorizationService{auth: authorizationSupport{gateway: authorization}}, nil
}
