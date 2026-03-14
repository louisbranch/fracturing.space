package app

// testGatewayBundle keeps the combined gateway seam available only to package
// tests so production app code stays on explicit capability config wiring.
type testGatewayBundle interface {
	CampaignCatalogReadGateway
	CampaignWorkspaceReadGateway
	CampaignGameReadGateway
	CampaignParticipantReadGateway
	CampaignCharacterReadGateway
	CampaignSessionReadGateway
	CampaignInviteReadGateway
	CampaignAutomationReadGateway
	CampaignCatalogMutationGateway
	CampaignConfigurationMutationGateway
	CampaignAutomationMutationGateway
	CampaignCharacterControlMutationGateway
	CampaignCharacterMutationGateway
	CampaignParticipantMutationGateway
	CampaignSessionMutationGateway
	CampaignInviteMutationGateway
	AuthorizationGateway
	BatchAuthorizationGateway
	CharacterCreationReadGateway
	CharacterCreationMutationGateway
}

// testServiceBundle keeps package tests convenient without reintroducing a
// production aggregate service surface.
type testServiceBundle struct {
	catalogService
	workspaceService
	gameService
	participantReadService
	participantMutationService
	automationReadService
	automationMutationService
	characterReadService
	characterControlService
	characterMutationService
	sessionReadService
	sessionMutationService
	inviteReadService
	inviteMutationService
	configurationService
	authorizationService
	creationPageService
	creationMutationService
}

func newService(gateway testGatewayBundle) testServiceBundle {
	if gateway == nil {
		gateway = NewUnavailableGateway()
	}
	auth := authorizationSupport{gateway: gateway}
	return testServiceBundle{
		catalogService: catalogService{
			read:     gateway,
			mutation: gateway,
		},
		workspaceService: workspaceService{
			read: gateway,
		},
		gameService: gameService{
			read: gateway,
		},
		participantReadService: participantReadService{
			read:               gateway,
			workspace:          gateway,
			batchAuthorization: gateway,
			auth:               auth,
		},
		participantMutationService: participantMutationService{
			read:      gateway,
			mutation:  gateway,
			workspace: gateway,
			auth:      auth,
		},
		automationReadService: automationReadService{
			participants: gateway,
			read:         gateway,
			auth:         auth,
		},
		automationMutationService: automationMutationService{
			participants: gateway,
			mutation:     gateway,
			auth:         auth,
		},
		characterReadService: characterReadService{
			read:               gateway,
			batchAuthorization: gateway,
			auth:               auth,
		},
		characterControlService: characterControlService{
			read:         gateway,
			mutation:     gateway,
			participants: gateway,
			sessions:     gateway,
			auth:         auth,
		},
		characterMutationService: characterMutationService{
			mutation: gateway,
			sessions: gateway,
			auth:     auth,
		},
		sessionReadService: sessionReadService{
			read: gateway,
		},
		sessionMutationService: sessionMutationService{
			mutation: gateway,
			auth:     auth,
		},
		inviteReadService: inviteReadService{
			read: gateway,
			auth: auth,
		},
		inviteMutationService: inviteMutationService{
			mutation: gateway,
			auth:     auth,
		},
		configurationService: configurationService{
			workspace: gateway,
			mutation:  gateway,
			auth:      auth,
		},
		authorizationService: authorizationService{
			auth: auth,
		},
		creationPageService: creationPageService{
			read: gateway,
		},
		creationMutationService: creationMutationService{
			read:     gateway,
			mutation: gateway,
			auth:     auth,
		},
	}
}
