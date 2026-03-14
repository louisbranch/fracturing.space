package app

// ParticipantReadServiceConfig keeps participant read/editor dependencies
// explicit.
type ParticipantReadServiceConfig struct {
	Read               CampaignParticipantReadGateway
	Workspace          CampaignWorkspaceReadGateway
	BatchAuthorization BatchAuthorizationGateway
}

// ParticipantMutationServiceConfig keeps participant mutation dependencies
// explicit.
type ParticipantMutationServiceConfig struct {
	Read      CampaignParticipantReadGateway
	Mutation  CampaignParticipantMutationGateway
	Workspace CampaignWorkspaceReadGateway
}

// AutomationReadServiceConfig keeps campaign automation read dependencies explicit.
type AutomationReadServiceConfig struct {
	Participants CampaignParticipantReadGateway
	Read         CampaignAutomationReadGateway
}

// AutomationMutationServiceConfig keeps campaign automation mutation dependencies explicit.
type AutomationMutationServiceConfig struct {
	Participants CampaignParticipantReadGateway
	Mutation     CampaignAutomationMutationGateway
}

// CharacterReadServiceConfig keeps character read dependencies explicit.
type CharacterReadServiceConfig struct {
	Read               CampaignCharacterReadGateway
	BatchAuthorization BatchAuthorizationGateway
}

// CharacterControlServiceConfig keeps character-control dependencies explicit.
type CharacterControlServiceConfig struct {
	Read         CampaignCharacterReadGateway
	Mutation     CampaignCharacterControlMutationGateway
	Participants CampaignParticipantReadGateway
	Sessions     CampaignSessionReadGateway
}

// CharacterMutationServiceConfig keeps character create/update/delete
// dependencies explicit.
type CharacterMutationServiceConfig struct {
	Mutation CampaignCharacterMutationGateway
	Sessions CampaignSessionReadGateway
}
