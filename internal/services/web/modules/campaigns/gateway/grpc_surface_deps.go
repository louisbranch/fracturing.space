package gateway

// PageGatewayDeps keeps shared detail-page gateway dependencies explicit.
type PageGatewayDeps struct {
	Workspace     WorkspaceReadDeps
	SessionRead   SessionReadDeps
	Authorization AuthorizationDeps
}

// CatalogGatewayDeps keeps catalog-only gateway dependencies explicit.
type CatalogGatewayDeps struct {
	Read     CatalogReadDeps
	Mutation CatalogMutationDeps
}

// StarterGatewayDeps keeps protected starter preview and launch dependencies explicit.
type StarterGatewayDeps struct {
	Starter StarterDeps
}

// OverviewGatewayDeps keeps overview/configuration/automation gateway
// dependencies explicit.
type OverviewGatewayDeps struct {
	Participants          ParticipantReadDeps
	Workspace             WorkspaceReadDeps
	Authorization         AuthorizationDeps
	AutomationRead        AutomationReadDeps
	AutomationMutation    AutomationMutationDeps
	ConfigurationMutation ConfigurationMutationDeps
}

// ParticipantGatewayDeps keeps participant route-surface gateway dependencies explicit.
type ParticipantGatewayDeps struct {
	Read          ParticipantReadDeps
	Mutation      ParticipantMutationDeps
	Workspace     WorkspaceReadDeps
	Authorization AuthorizationDeps
}

// CharacterGatewayDeps keeps character/ownership/creation gateway dependencies explicit.
type CharacterGatewayDeps struct {
	Read             CharacterReadDeps
	Ownership        CharacterOwnershipMutationDeps
	Mutation         CharacterMutationDeps
	Participants     ParticipantReadDeps
	Sessions         SessionReadDeps
	Authorization    AuthorizationDeps
	CreationRead     CharacterCreationReadDeps
	CreationMutation CharacterCreationMutationDeps
}

// SessionGatewayDeps keeps session lifecycle gateway dependencies explicit.
type SessionGatewayDeps struct {
	Mutation SessionMutationDeps
}

// InviteGatewayDeps keeps invite/search gateway dependencies explicit.
type InviteGatewayDeps struct {
	Read          InviteReadDeps
	Mutation      InviteMutationDeps
	Participants  ParticipantReadDeps
	Authorization AuthorizationDeps
}

// GRPCGatewayDeps keeps startup and test dependency grouping explicit by owned
// route surface instead of one flat capability bag.
type GRPCGatewayDeps struct {
	Page         PageGatewayDeps
	Catalog      CatalogGatewayDeps
	Starter      StarterGatewayDeps
	Overview     OverviewGatewayDeps
	Participants ParticipantGatewayDeps
	Characters   CharacterGatewayDeps
	Sessions     SessionGatewayDeps
	Invites      InviteGatewayDeps
	AssetBaseURL string
}
