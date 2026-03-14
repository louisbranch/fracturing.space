package app

// catalogService keeps catalog reads and mutations local to catalog ownership.
type catalogService struct {
	read     CampaignCatalogReadGateway
	mutation CampaignCatalogMutationGateway
}

// workspaceService keeps campaign workspace summaries on their own read seam.
type workspaceService struct {
	read CampaignWorkspaceReadGateway
}

// gameService keeps game-surface reads isolated from generic workspace data.
type gameService struct {
	read CampaignGameReadGateway
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
// gateway seams.
func NewCatalogService(config CatalogServiceConfig) CampaignCatalogService {
	if config.Read == nil || config.Mutation == nil {
		return nil
	}
	return catalogService{read: config.Read, mutation: config.Mutation}
}

// NewWorkspaceService constructs the workspace-read service surface from
// explicit gateway seams.
func NewWorkspaceService(config WorkspaceServiceConfig) CampaignWorkspaceService {
	if config.Read == nil {
		return nil
	}
	return workspaceService{read: config.Read}
}

// NewGameService constructs the game-surface service surface from explicit
// gateway seams.
func NewGameService(config GameServiceConfig) CampaignGameService {
	if config.Read == nil {
		return nil
	}
	return gameService{read: config.Read}
}

// NewAuthorizationService constructs the authorization service surface from
// explicit gateway seams.
func NewAuthorizationService(authorization AuthorizationGateway) CampaignAuthorizationService {
	if authorization == nil {
		return nil
	}
	return authorizationService{auth: authorizationSupport{gateway: authorization}}
}
