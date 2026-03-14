package gateway

import (
	"context"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"golang.org/x/text/language"
)

// testGatewayBundle keeps the combined gateway seam available only to gateway
// package tests. Production code should stay on explicit capability
// constructors.
type testGatewayBundle interface {
	campaignapp.CampaignCatalogReadGateway
	campaignapp.CampaignWorkspaceReadGateway
	campaignapp.CampaignGameReadGateway
	campaignapp.CampaignParticipantReadGateway
	campaignapp.CampaignCharacterReadGateway
	campaignapp.CampaignSessionReadGateway
	campaignapp.CampaignInviteReadGateway
	campaignapp.CampaignAutomationReadGateway
	campaignapp.CampaignCatalogMutationGateway
	campaignapp.CampaignConfigurationMutationGateway
	campaignapp.CampaignAutomationMutationGateway
	campaignapp.CampaignCharacterControlMutationGateway
	campaignapp.CampaignCharacterMutationGateway
	campaignapp.CampaignParticipantMutationGateway
	campaignapp.CampaignSessionMutationGateway
	campaignapp.CampaignInviteMutationGateway
	campaignapp.AuthorizationGateway
	campaignapp.BatchAuthorizationGateway
	campaignapp.CharacterCreationReadGateway
	campaignapp.CharacterCreationMutationGateway
}

// GRPCGatewayReadDeps exists only in tests so package tests can still express
// broad read fixtures without reintroducing that dependency bag to production.
type GRPCGatewayReadDeps struct {
	Campaign           CampaignReadClient
	Communication      CommunicationClient
	Agent              AgentClient
	Participant        ParticipantReadClient
	Character          CharacterReadClient
	DaggerheartContent DaggerheartContentClient
	DaggerheartAsset   DaggerheartAssetClient
	Session            SessionReadClient
	Invite             InviteReadClient
	Social             SocialClient
}

// GRPCGatewayCreationReadDeps exists only in tests for concise workflow-read fixtures.
type GRPCGatewayCreationReadDeps struct {
	Character          CharacterReadClient
	DaggerheartContent DaggerheartContentClient
	DaggerheartAsset   DaggerheartAssetClient
}

// GRPCGatewayMutationDeps exists only in tests so package tests can still
// express broad mutation fixtures without reintroducing that dependency bag to
// production.
type GRPCGatewayMutationDeps struct {
	Campaign         CampaignMutationClient
	Participant      ParticipantMutationClient
	CharacterControl CharacterMutationClient
	Character        CharacterMutationClient
	Session          SessionMutationClient
	Invite           InviteMutationClient
	Auth             AuthClient
}

// GRPCGatewayCreationMutationDeps exists only in tests for concise workflow-mutation fixtures.
type GRPCGatewayCreationMutationDeps struct {
	Character CharacterMutationClient
}

// GRPCGatewayAuthorizationDeps exists only in tests for concise authz fixtures.
type GRPCGatewayAuthorizationDeps struct {
	Client AuthorizationClient
}

// GRPCGateway exists only in tests so gateway package tests can continue to
// compose capability adapters from grouped deps without reintroducing a
// production aggregate adapter type.
type GRPCGateway struct {
	Read             GRPCGatewayReadDeps
	CreationRead     GRPCGatewayCreationReadDeps
	Mutation         GRPCGatewayMutationDeps
	CreationMutation GRPCGatewayCreationMutationDeps
	Authorization    GRPCGatewayAuthorizationDeps
	AssetBaseURL     string
}

func NewGRPCGateway(deps GRPCGatewayDeps) testGatewayBundle {
	if deps.CatalogRead.Campaign == nil || deps.GameRead.Communication == nil || deps.ParticipantRead.Participant == nil ||
		deps.CharacterRead.Character == nil || deps.CharacterRead.DaggerheartContent == nil ||
		deps.SessionRead.Session == nil || deps.SessionRead.Campaign == nil ||
		deps.InviteRead.Invite == nil || deps.InviteRead.Participant == nil || deps.InviteRead.Social == nil || deps.InviteRead.Auth == nil ||
		deps.AutomationRead.Agent == nil ||
		deps.CatalogMutation.Campaign == nil || deps.ParticipantMutate.Participant == nil ||
		deps.CharacterControl.Character == nil || deps.CharacterMutate.Character == nil ||
		deps.SessionMutate.Session == nil || deps.InviteMutate.Invite == nil || deps.InviteMutate.Auth == nil ||
		deps.ConfigMutate.Campaign == nil || deps.AutomationMutate.Campaign == nil ||
		deps.Authorization.Client == nil || deps.CreationRead.Character == nil ||
		deps.CreationRead.DaggerheartContent == nil || deps.CreationRead.DaggerheartAsset == nil ||
		deps.CreationMutation.Character == nil {
		return campaignapp.NewUnavailableGateway()
	}
	return GRPCGateway{
		Read: GRPCGatewayReadDeps{
			Campaign:           deps.CatalogRead.Campaign,
			Communication:      deps.GameRead.Communication,
			Agent:              deps.AutomationRead.Agent,
			Participant:        deps.ParticipantRead.Participant,
			Character:          deps.CharacterRead.Character,
			DaggerheartContent: deps.CharacterRead.DaggerheartContent,
			DaggerheartAsset:   deps.CreationRead.DaggerheartAsset,
			Session:            deps.SessionRead.Session,
			Invite:             deps.InviteRead.Invite,
			Social:             deps.InviteRead.Social,
		},
		CreationRead: GRPCGatewayCreationReadDeps{
			Character:          deps.CreationRead.Character,
			DaggerheartContent: deps.CreationRead.DaggerheartContent,
			DaggerheartAsset:   deps.CreationRead.DaggerheartAsset,
		},
		Mutation: GRPCGatewayMutationDeps{
			Campaign:         deps.CatalogMutation.Campaign,
			Participant:      deps.ParticipantMutate.Participant,
			CharacterControl: deps.CharacterControl.Character,
			Character:        deps.CharacterMutate.Character,
			Session:          deps.SessionMutate.Session,
			Invite:           deps.InviteMutate.Invite,
			Auth:             deps.InviteMutate.Auth,
		},
		CreationMutation: GRPCGatewayCreationMutationDeps{
			Character: deps.CreationMutation.Character,
		},
		Authorization: GRPCGatewayAuthorizationDeps{
			Client: deps.Authorization.Client,
		},
		AssetBaseURL: deps.AssetBaseURL,
	}
}

func (g GRPCGateway) unavailable() testGatewayBundle {
	return campaignapp.NewUnavailableGateway()
}

func (g GRPCGateway) catalogRead() campaignapp.CampaignCatalogReadGateway {
	return catalogReadGateway{
		read:         CatalogReadDeps{Campaign: g.Read.Campaign},
		assetBaseURL: g.AssetBaseURL,
	}
}

func (g GRPCGateway) catalogMutation() campaignapp.CampaignCatalogMutationGateway {
	return catalogMutationGateway{mutation: CatalogMutationDeps{Campaign: g.Mutation.Campaign}}
}

func (g GRPCGateway) workspaceRead() campaignapp.CampaignWorkspaceReadGateway {
	return workspaceReadGateway{
		read:         WorkspaceReadDeps{Campaign: g.Read.Campaign},
		assetBaseURL: g.AssetBaseURL,
	}
}

func (g GRPCGateway) gameRead() campaignapp.CampaignGameReadGateway {
	return gameReadGateway{read: GameReadDeps{Communication: g.Read.Communication}}
}

func (g GRPCGateway) participantRead() campaignapp.CampaignParticipantReadGateway {
	return participantReadGateway{
		read:         ParticipantReadDeps{Participant: g.Read.Participant},
		assetBaseURL: g.AssetBaseURL,
	}
}

func (g GRPCGateway) participantMutation() campaignapp.CampaignParticipantMutationGateway {
	return participantMutationGateway{mutation: ParticipantMutationDeps{Participant: g.Mutation.Participant}}
}

func (g GRPCGateway) characterRead() campaignapp.CampaignCharacterReadGateway {
	return characterReadGateway{
		read: CharacterReadDeps{
			Participant:        g.Read.Participant,
			Character:          g.Read.Character,
			DaggerheartContent: g.Read.DaggerheartContent,
		},
		assetBaseURL: g.AssetBaseURL,
	}
}

func (g GRPCGateway) characterMutation() campaignapp.CampaignCharacterMutationGateway {
	return characterMutationGateway{mutation: CharacterMutationDeps{Character: g.Mutation.Character}}
}

func (g GRPCGateway) characterControlMutation() campaignapp.CampaignCharacterControlMutationGateway {
	return characterControlMutationGateway{mutation: CharacterControlMutationDeps{Character: g.Mutation.CharacterControl}}
}

func (g GRPCGateway) sessionRead() campaignapp.CampaignSessionReadGateway {
	return sessionReadGateway{read: SessionReadDeps{Campaign: g.Read.Campaign, Session: g.Read.Session}}
}

func (g GRPCGateway) sessionMutation() campaignapp.CampaignSessionMutationGateway {
	return sessionMutationGateway{mutation: SessionMutationDeps{Session: g.Mutation.Session}}
}

func (g GRPCGateway) inviteRead() campaignapp.CampaignInviteReadGateway {
	return inviteReadGateway{read: InviteReadDeps{
		Invite:      g.Read.Invite,
		Participant: g.Read.Participant,
		Social:      g.Read.Social,
		Auth:        g.Mutation.Auth,
	}}
}

func (g GRPCGateway) inviteMutation() campaignapp.CampaignInviteMutationGateway {
	return inviteMutationGateway{mutation: InviteMutationDeps{Invite: g.Mutation.Invite, Auth: g.Mutation.Auth}}
}

func (g GRPCGateway) configurationMutation() campaignapp.CampaignConfigurationMutationGateway {
	return configurationMutationGateway{mutation: ConfigurationMutationDeps{Campaign: g.Mutation.Campaign}}
}

func (g GRPCGateway) automationRead() campaignapp.CampaignAutomationReadGateway {
	return automationReadGateway{read: AutomationReadDeps{Agent: g.Read.Agent}}
}

func (g GRPCGateway) automationMutation() campaignapp.CampaignAutomationMutationGateway {
	return automationMutationGateway{mutation: AutomationMutationDeps{Campaign: g.Mutation.Campaign}}
}

func (g GRPCGateway) authorizationGateway() campaignapp.AuthorizationGateway {
	return authorizationGateway{authorization: AuthorizationDeps{Client: g.Authorization.Client}}
}

func (g GRPCGateway) batchAuthorizationGateway() campaignapp.BatchAuthorizationGateway {
	return batchAuthorizationGateway{authorization: AuthorizationDeps{Client: g.Authorization.Client}}
}

func (g GRPCGateway) creationRead() campaignapp.CharacterCreationReadGateway {
	return characterCreationReadGateway{read: CharacterCreationReadDeps{
		Character:          g.CreationRead.Character,
		DaggerheartContent: g.CreationRead.DaggerheartContent,
		DaggerheartAsset:   g.CreationRead.DaggerheartAsset,
	}, assetBaseURL: g.AssetBaseURL}
}

func (g GRPCGateway) creationMutation() campaignapp.CharacterCreationMutationGateway {
	return characterCreationMutationGateway{mutation: CharacterCreationMutationDeps{Character: g.CreationMutation.Character}}
}

func (g GRPCGateway) ListCampaigns(ctx context.Context) ([]campaignapp.CampaignSummary, error) {
	if gateway := g.catalogRead(); gateway != nil {
		return gateway.ListCampaigns(ctx)
	}
	return g.unavailable().ListCampaigns(ctx)
}

func (g GRPCGateway) CampaignName(ctx context.Context, campaignID string) (string, error) {
	if gateway := g.workspaceRead(); gateway != nil {
		return gateway.CampaignName(ctx, campaignID)
	}
	return g.unavailable().CampaignName(ctx, campaignID)
}

func (g GRPCGateway) CampaignWorkspace(ctx context.Context, campaignID string) (campaignapp.CampaignWorkspace, error) {
	if gateway := g.workspaceRead(); gateway != nil {
		return gateway.CampaignWorkspace(ctx, campaignID)
	}
	return g.unavailable().CampaignWorkspace(ctx, campaignID)
}

func (g GRPCGateway) CampaignGameSurface(ctx context.Context, campaignID string) (campaignapp.CampaignGameSurface, error) {
	if gateway := g.gameRead(); gateway != nil {
		return gateway.CampaignGameSurface(ctx, campaignID)
	}
	return g.unavailable().CampaignGameSurface(ctx, campaignID)
}

func (g GRPCGateway) CampaignParticipants(ctx context.Context, campaignID string) ([]campaignapp.CampaignParticipant, error) {
	if gateway := g.participantRead(); gateway != nil {
		return gateway.CampaignParticipants(ctx, campaignID)
	}
	return g.unavailable().CampaignParticipants(ctx, campaignID)
}

func (g GRPCGateway) CampaignParticipant(ctx context.Context, campaignID string, participantID string) (campaignapp.CampaignParticipant, error) {
	if gateway := g.participantRead(); gateway != nil {
		return gateway.CampaignParticipant(ctx, campaignID, participantID)
	}
	return g.unavailable().CampaignParticipant(ctx, campaignID, participantID)
}

func (g GRPCGateway) CampaignCharacters(ctx context.Context, campaignID string, options campaignapp.CharacterReadContext) ([]campaignapp.CampaignCharacter, error) {
	if gateway := g.characterRead(); gateway != nil {
		return gateway.CampaignCharacters(ctx, campaignID, options)
	}
	return g.unavailable().CampaignCharacters(ctx, campaignID, options)
}

func (g GRPCGateway) CampaignCharacter(ctx context.Context, campaignID string, characterID string, options campaignapp.CharacterReadContext) (campaignapp.CampaignCharacter, error) {
	if gateway := g.characterRead(); gateway != nil {
		return gateway.CampaignCharacter(ctx, campaignID, characterID, options)
	}
	return g.unavailable().CampaignCharacter(ctx, campaignID, characterID, options)
}

func (g GRPCGateway) CampaignSessions(ctx context.Context, campaignID string) ([]campaignapp.CampaignSession, error) {
	if gateway := g.sessionRead(); gateway != nil {
		return gateway.CampaignSessions(ctx, campaignID)
	}
	return g.unavailable().CampaignSessions(ctx, campaignID)
}

func (g GRPCGateway) CampaignSessionReadiness(ctx context.Context, campaignID string, locale language.Tag) (campaignapp.CampaignSessionReadiness, error) {
	if gateway := g.sessionRead(); gateway != nil {
		return gateway.CampaignSessionReadiness(ctx, campaignID, locale)
	}
	return g.unavailable().CampaignSessionReadiness(ctx, campaignID, locale)
}

func (g GRPCGateway) CampaignInvites(ctx context.Context, campaignID string) ([]campaignapp.CampaignInvite, error) {
	if gateway := g.inviteRead(); gateway != nil {
		return gateway.CampaignInvites(ctx, campaignID)
	}
	return g.unavailable().CampaignInvites(ctx, campaignID)
}

func (g GRPCGateway) SearchInviteUsers(ctx context.Context, input campaignapp.SearchInviteUsersInput) ([]campaignapp.InviteUserSearchResult, error) {
	if gateway := g.inviteRead(); gateway != nil {
		return gateway.SearchInviteUsers(ctx, input)
	}
	return g.unavailable().SearchInviteUsers(ctx, input)
}

func (g GRPCGateway) CampaignAIAgents(ctx context.Context) ([]campaignapp.CampaignAIAgentOption, error) {
	if gateway := g.automationRead(); gateway != nil {
		return gateway.CampaignAIAgents(ctx)
	}
	return g.unavailable().CampaignAIAgents(ctx)
}

func (g GRPCGateway) CharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (campaignapp.CampaignCharacterCreationProgress, error) {
	if gateway := g.creationRead(); gateway != nil {
		return gateway.CharacterCreationProgress(ctx, campaignID, characterID)
	}
	return g.unavailable().CharacterCreationProgress(ctx, campaignID, characterID)
}

func (g GRPCGateway) CharacterCreationCatalog(ctx context.Context, locale language.Tag) (campaignapp.CampaignCharacterCreationCatalog, error) {
	if gateway := g.creationRead(); gateway != nil {
		return gateway.CharacterCreationCatalog(ctx, locale)
	}
	return g.unavailable().CharacterCreationCatalog(ctx, locale)
}

func (g GRPCGateway) CharacterCreationProfile(ctx context.Context, campaignID string, characterID string) (campaignapp.CampaignCharacterCreationProfile, error) {
	if gateway := g.creationRead(); gateway != nil {
		return gateway.CharacterCreationProfile(ctx, campaignID, characterID)
	}
	return g.unavailable().CharacterCreationProfile(ctx, campaignID, characterID)
}

func (g GRPCGateway) CreateCampaign(ctx context.Context, input campaignapp.CreateCampaignInput) (campaignapp.CreateCampaignResult, error) {
	if gateway := g.catalogMutation(); gateway != nil {
		return gateway.CreateCampaign(ctx, input)
	}
	return g.unavailable().CreateCampaign(ctx, input)
}

func (g GRPCGateway) UpdateCampaign(ctx context.Context, campaignID string, input campaignapp.UpdateCampaignInput) error {
	if gateway := g.configurationMutation(); gateway != nil {
		return gateway.UpdateCampaign(ctx, campaignID, input)
	}
	return g.unavailable().UpdateCampaign(ctx, campaignID, input)
}

func (g GRPCGateway) UpdateCampaignAIBinding(ctx context.Context, campaignID string, input campaignapp.UpdateCampaignAIBindingInput) error {
	if gateway := g.automationMutation(); gateway != nil {
		return gateway.UpdateCampaignAIBinding(ctx, campaignID, input)
	}
	return g.unavailable().UpdateCampaignAIBinding(ctx, campaignID, input)
}

func (g GRPCGateway) CreateCharacter(ctx context.Context, campaignID string, input campaignapp.CreateCharacterInput) (campaignapp.CreateCharacterResult, error) {
	if gateway := g.characterMutation(); gateway != nil {
		return gateway.CreateCharacter(ctx, campaignID, input)
	}
	return g.unavailable().CreateCharacter(ctx, campaignID, input)
}

func (g GRPCGateway) UpdateCharacter(ctx context.Context, campaignID string, characterID string, input campaignapp.UpdateCharacterInput) error {
	if gateway := g.characterMutation(); gateway != nil {
		return gateway.UpdateCharacter(ctx, campaignID, characterID, input)
	}
	return g.unavailable().UpdateCharacter(ctx, campaignID, characterID, input)
}

func (g GRPCGateway) DeleteCharacter(ctx context.Context, campaignID string, characterID string) error {
	if gateway := g.characterMutation(); gateway != nil {
		return gateway.DeleteCharacter(ctx, campaignID, characterID)
	}
	return g.unavailable().DeleteCharacter(ctx, campaignID, characterID)
}

func (g GRPCGateway) SetCharacterController(ctx context.Context, campaignID string, characterID string, participantID string) error {
	if gateway := g.characterControlMutation(); gateway != nil {
		return gateway.SetCharacterController(ctx, campaignID, characterID, participantID)
	}
	return g.unavailable().SetCharacterController(ctx, campaignID, characterID, participantID)
}

func (g GRPCGateway) ClaimCharacterControl(ctx context.Context, campaignID string, characterID string) error {
	if gateway := g.characterControlMutation(); gateway != nil {
		return gateway.ClaimCharacterControl(ctx, campaignID, characterID)
	}
	return g.unavailable().ClaimCharacterControl(ctx, campaignID, characterID)
}

func (g GRPCGateway) ReleaseCharacterControl(ctx context.Context, campaignID string, characterID string) error {
	if gateway := g.characterControlMutation(); gateway != nil {
		return gateway.ReleaseCharacterControl(ctx, campaignID, characterID)
	}
	return g.unavailable().ReleaseCharacterControl(ctx, campaignID, characterID)
}

func (g GRPCGateway) CreateParticipant(ctx context.Context, campaignID string, input campaignapp.CreateParticipantInput) (campaignapp.CreateParticipantResult, error) {
	if gateway := g.participantMutation(); gateway != nil {
		return gateway.CreateParticipant(ctx, campaignID, input)
	}
	return g.unavailable().CreateParticipant(ctx, campaignID, input)
}

func (g GRPCGateway) UpdateParticipant(ctx context.Context, campaignID string, input campaignapp.UpdateParticipantInput) error {
	if gateway := g.participantMutation(); gateway != nil {
		return gateway.UpdateParticipant(ctx, campaignID, input)
	}
	return g.unavailable().UpdateParticipant(ctx, campaignID, input)
}

func (g GRPCGateway) StartSession(ctx context.Context, campaignID string, input campaignapp.StartSessionInput) error {
	if gateway := g.sessionMutation(); gateway != nil {
		return gateway.StartSession(ctx, campaignID, input)
	}
	return g.unavailable().StartSession(ctx, campaignID, input)
}

func (g GRPCGateway) EndSession(ctx context.Context, campaignID string, input campaignapp.EndSessionInput) error {
	if gateway := g.sessionMutation(); gateway != nil {
		return gateway.EndSession(ctx, campaignID, input)
	}
	return g.unavailable().EndSession(ctx, campaignID, input)
}

func (g GRPCGateway) CreateInvite(ctx context.Context, campaignID string, input campaignapp.CreateInviteInput) error {
	if gateway := g.inviteMutation(); gateway != nil {
		return gateway.CreateInvite(ctx, campaignID, input)
	}
	return g.unavailable().CreateInvite(ctx, campaignID, input)
}

func (g GRPCGateway) RevokeInvite(ctx context.Context, campaignID string, input campaignapp.RevokeInviteInput) error {
	if gateway := g.inviteMutation(); gateway != nil {
		return gateway.RevokeInvite(ctx, campaignID, input)
	}
	return g.unavailable().RevokeInvite(ctx, campaignID, input)
}

func (g GRPCGateway) ApplyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, input *campaignapp.CampaignCharacterCreationStepInput) error {
	if gateway := g.creationMutation(); gateway != nil {
		return gateway.ApplyCharacterCreationStep(ctx, campaignID, characterID, input)
	}
	return g.unavailable().ApplyCharacterCreationStep(ctx, campaignID, characterID, input)
}

func (g GRPCGateway) ResetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	if gateway := g.creationMutation(); gateway != nil {
		return gateway.ResetCharacterCreationWorkflow(ctx, campaignID, characterID)
	}
	return g.unavailable().ResetCharacterCreationWorkflow(ctx, campaignID, characterID)
}

func (g GRPCGateway) CanCampaignAction(
	ctx context.Context,
	campaignID string,
	action campaignapp.AuthorizationAction,
	resource campaignapp.AuthorizationResource,
	target *campaignapp.AuthorizationTarget,
) (campaignapp.AuthorizationDecision, error) {
	if gateway := g.authorizationGateway(); gateway != nil {
		return gateway.CanCampaignAction(ctx, campaignID, action, resource, target)
	}
	return g.unavailable().CanCampaignAction(ctx, campaignID, action, resource, target)
}

func (g GRPCGateway) BatchCanCampaignAction(ctx context.Context, campaignID string, checks []campaignapp.AuthorizationCheck) ([]campaignapp.AuthorizationDecision, error) {
	if gateway := g.batchAuthorizationGateway(); gateway != nil {
		return gateway.BatchCanCampaignAction(ctx, campaignID, checks)
	}
	return g.unavailable().BatchCanCampaignAction(ctx, campaignID, checks)
}
