package app

// ServiceConfig keeps constructor dependencies explicit by capability.
type ServiceConfig struct {
	Catalog             CatalogServiceConfig
	Starter             StarterServiceConfig
	Workspace           WorkspaceServiceConfig
	Game                GameServiceConfig
	ParticipantRead     ParticipantReadServiceConfig
	ParticipantMutation ParticipantMutationServiceConfig
	CharacterRead       CharacterReadServiceConfig
	CharacterControl    CharacterControlServiceConfig
	CharacterMutation   CharacterMutationServiceConfig
	SessionRead         SessionReadServiceConfig
	SessionMutation     SessionMutationServiceConfig
	InviteRead          InviteReadServiceConfig
	InviteMutation      InviteMutationServiceConfig
	Configuration       ConfigurationServiceConfig
	AutomationRead      AutomationReadServiceConfig
	AutomationMutation  AutomationMutationServiceConfig
	Creation            CharacterCreationServiceConfig
	Authorization       AuthorizationGateway
}
