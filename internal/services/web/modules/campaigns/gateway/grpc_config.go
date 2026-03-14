package gateway

import campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"

// CatalogReadDeps keeps campaign catalog query dependencies explicit.
type CatalogReadDeps struct {
	Campaign CampaignReadClient
}

// CatalogMutationDeps keeps campaign catalog mutation dependencies explicit.
type CatalogMutationDeps struct {
	Campaign CampaignMutationClient
}

// WorkspaceReadDeps keeps workspace query dependencies explicit.
type WorkspaceReadDeps struct {
	Campaign CampaignReadClient
}

// GameReadDeps keeps game-surface query dependencies explicit.
type GameReadDeps struct {
	Communication CommunicationClient
}

// ParticipantReadDeps keeps participant query dependencies explicit.
type ParticipantReadDeps struct {
	Participant ParticipantReadClient
}

// ParticipantMutationDeps keeps participant mutation dependencies explicit.
type ParticipantMutationDeps struct {
	Participant ParticipantMutationClient
}

// CharacterReadDeps keeps character query dependencies explicit.
type CharacterReadDeps struct {
	Character          CharacterReadClient
	Participant        ParticipantReadClient
	DaggerheartContent DaggerheartContentClient
}

// CharacterMutationDeps keeps character mutation dependencies explicit.
type CharacterMutationDeps struct {
	Character CharacterMutationClient
}

// CharacterControlMutationDeps keeps character-control mutation dependencies
// explicit.
type CharacterControlMutationDeps struct {
	Character CharacterMutationClient
}

// SessionReadDeps keeps session query dependencies explicit.
type SessionReadDeps struct {
	Session  SessionReadClient
	Campaign CampaignReadClient
}

// SessionMutationDeps keeps session mutation dependencies explicit.
type SessionMutationDeps struct {
	Session SessionMutationClient
}

// InviteReadDeps keeps invite read/search dependencies explicit.
type InviteReadDeps struct {
	Invite      InviteReadClient
	Participant ParticipantReadClient
	Social      SocialClient
	Auth        AuthClient
}

// InviteMutationDeps keeps invite mutation dependencies explicit.
type InviteMutationDeps struct {
	Invite InviteMutationClient
	Auth   AuthClient
}

// ConfigurationMutationDeps keeps campaign settings mutation dependencies explicit.
type ConfigurationMutationDeps struct {
	Campaign CampaignMutationClient
}

// AutomationReadDeps keeps automation query dependencies explicit.
type AutomationReadDeps struct {
	Agent AgentClient
}

// AutomationMutationDeps keeps automation mutation dependencies explicit.
type AutomationMutationDeps struct {
	Campaign CampaignMutationClient
}

// AuthorizationDeps keeps authorization dependencies explicit.
type AuthorizationDeps struct {
	Client AuthorizationClient
}

// CharacterCreationReadDeps keeps creation workflow read dependencies explicit.
type CharacterCreationReadDeps struct {
	Character          CharacterReadClient
	DaggerheartContent DaggerheartContentClient
	DaggerheartAsset   DaggerheartAssetClient
}

// CharacterCreationMutationDeps keeps creation workflow mutation dependencies explicit.
type CharacterCreationMutationDeps struct {
	Character CharacterMutationClient
}

// GRPCGatewayDeps keeps startup and test dependency grouping explicit without
// reintroducing flat read/mutation dependency bags.
type GRPCGatewayDeps struct {
	CatalogRead       CatalogReadDeps
	CatalogMutation   CatalogMutationDeps
	WorkspaceRead     WorkspaceReadDeps
	GameRead          GameReadDeps
	ParticipantRead   ParticipantReadDeps
	ParticipantMutate ParticipantMutationDeps
	CharacterRead     CharacterReadDeps
	CharacterControl  CharacterControlMutationDeps
	CharacterMutate   CharacterMutationDeps
	SessionRead       SessionReadDeps
	SessionMutate     SessionMutationDeps
	InviteRead        InviteReadDeps
	InviteMutate      InviteMutationDeps
	ConfigMutate      ConfigurationMutationDeps
	AutomationRead    AutomationReadDeps
	AutomationMutate  AutomationMutationDeps
	Authorization     AuthorizationDeps
	CreationRead      CharacterCreationReadDeps
	CreationMutation  CharacterCreationMutationDeps
	AssetBaseURL      string
}

// catalogReadGateway maps campaign catalog reads from the campaign backend only.
type catalogReadGateway struct {
	read         CatalogReadDeps
	assetBaseURL string
}

// catalogMutationGateway maps campaign catalog mutations without widening read deps.
type catalogMutationGateway struct {
	mutation CatalogMutationDeps
}

// workspaceReadGateway maps workspace summary reads from campaign state.
type workspaceReadGateway struct {
	read         WorkspaceReadDeps
	assetBaseURL string
}

// gameReadGateway maps game-surface reads from communication state.
type gameReadGateway struct {
	read GameReadDeps
}

// participantReadGateway maps participant reads and view formatting inputs.
type participantReadGateway struct {
	read         ParticipantReadDeps
	assetBaseURL string
}

// participantMutationGateway maps participant mutations without carrying unrelated clients.
type participantMutationGateway struct {
	mutation ParticipantMutationDeps
}

// characterReadGateway maps character list/entity reads and derived view data.
type characterReadGateway struct {
	read         CharacterReadDeps
	assetBaseURL string
}

// characterMutationGateway maps character mutations only.
type characterMutationGateway struct {
	mutation CharacterMutationDeps
}

// characterControlMutationGateway maps character controller mutations only.
type characterControlMutationGateway struct {
	mutation CharacterControlMutationDeps
}

// sessionReadGateway maps session reads independently of session mutations.
type sessionReadGateway struct {
	read SessionReadDeps
}

// sessionMutationGateway maps session lifecycle mutations only.
type sessionMutationGateway struct {
	mutation SessionMutationDeps
}

// inviteReadGateway maps invite reads and invite-search side effects from owned deps.
type inviteReadGateway struct {
	read InviteReadDeps
}

// inviteMutationGateway maps invite mutations without widening read authority.
type inviteMutationGateway struct {
	mutation InviteMutationDeps
}

// configurationMutationGateway maps campaign settings mutations only.
type configurationMutationGateway struct {
	mutation ConfigurationMutationDeps
}

// automationReadGateway maps campaign automation editor reads.
type automationReadGateway struct {
	read AutomationReadDeps
}

// automationMutationGateway maps campaign automation edits.
type automationMutationGateway struct {
	mutation AutomationMutationDeps
}

// authorizationGateway maps unary authorization checks for mutation guards.
type authorizationGateway struct {
	authorization AuthorizationDeps
}

// batchAuthorizationGateway maps row-hydration authorization checks separately from unary guards.
type batchAuthorizationGateway struct {
	authorization AuthorizationDeps
}

// characterCreationReadGateway maps character-creation workflow reads.
type characterCreationReadGateway struct {
	read         CharacterCreationReadDeps
	assetBaseURL string
}

// characterCreationMutationGateway maps character-creation workflow mutations.
type characterCreationMutationGateway struct {
	mutation CharacterCreationMutationDeps
}

// NewCatalogReadGateway builds the campaign catalog read adapter from explicit
// dependencies.
func NewCatalogReadGateway(readDeps CatalogReadDeps, assetBaseURL string) campaignapp.CampaignCatalogReadGateway {
	if readDeps.Campaign == nil {
		return nil
	}
	return catalogReadGateway{
		read:         readDeps,
		assetBaseURL: assetBaseURL,
	}
}

// NewCatalogMutationGateway builds the campaign catalog mutation adapter from
// explicit dependencies.
func NewCatalogMutationGateway(mutationDeps CatalogMutationDeps) campaignapp.CampaignCatalogMutationGateway {
	if mutationDeps.Campaign == nil {
		return nil
	}
	return catalogMutationGateway{mutation: mutationDeps}
}

// NewWorkspaceReadGateway builds the workspace read adapter from explicit
// dependencies.
func NewWorkspaceReadGateway(readDeps WorkspaceReadDeps, assetBaseURL string) campaignapp.CampaignWorkspaceReadGateway {
	if readDeps.Campaign == nil {
		return nil
	}
	return workspaceReadGateway{
		read:         readDeps,
		assetBaseURL: assetBaseURL,
	}
}

// NewGameReadGateway builds the game-surface read adapter from explicit
// dependencies.
func NewGameReadGateway(readDeps GameReadDeps) campaignapp.CampaignGameReadGateway {
	if readDeps.Communication == nil {
		return nil
	}
	return gameReadGateway{read: readDeps}
}

// NewParticipantReadGateway builds the participant read adapter from explicit
// dependencies.
func NewParticipantReadGateway(readDeps ParticipantReadDeps, assetBaseURL string) campaignapp.CampaignParticipantReadGateway {
	if readDeps.Participant == nil {
		return nil
	}
	return participantReadGateway{
		read:         readDeps,
		assetBaseURL: assetBaseURL,
	}
}

// NewParticipantMutationGateway builds the participant mutation adapter from
// explicit dependencies.
func NewParticipantMutationGateway(mutationDeps ParticipantMutationDeps) campaignapp.CampaignParticipantMutationGateway {
	if mutationDeps.Participant == nil {
		return nil
	}
	return participantMutationGateway{mutation: mutationDeps}
}

// NewCharacterReadGateway builds the character read adapter from explicit
// dependencies.
func NewCharacterReadGateway(readDeps CharacterReadDeps, assetBaseURL string) campaignapp.CampaignCharacterReadGateway {
	if readDeps.Character == nil || readDeps.Participant == nil || readDeps.DaggerheartContent == nil {
		return nil
	}
	return characterReadGateway{
		read:         readDeps,
		assetBaseURL: assetBaseURL,
	}
}

// NewCharacterMutationGateway builds the character mutation adapter from
// explicit dependencies.
func NewCharacterMutationGateway(mutationDeps CharacterMutationDeps) campaignapp.CampaignCharacterMutationGateway {
	if mutationDeps.Character == nil {
		return nil
	}
	return characterMutationGateway{mutation: mutationDeps}
}

// NewCharacterControlMutationGateway builds the character-control mutation
// adapter from explicit dependencies.
func NewCharacterControlMutationGateway(mutationDeps CharacterControlMutationDeps) campaignapp.CampaignCharacterControlMutationGateway {
	if mutationDeps.Character == nil {
		return nil
	}
	return characterControlMutationGateway{mutation: mutationDeps}
}

// NewSessionReadGateway builds the session read adapter from explicit
// dependencies.
func NewSessionReadGateway(readDeps SessionReadDeps) campaignapp.CampaignSessionReadGateway {
	if readDeps.Session == nil || readDeps.Campaign == nil {
		return nil
	}
	return sessionReadGateway{read: readDeps}
}

// NewSessionMutationGateway builds the session mutation adapter from explicit
// dependencies.
func NewSessionMutationGateway(mutationDeps SessionMutationDeps) campaignapp.CampaignSessionMutationGateway {
	if mutationDeps.Session == nil {
		return nil
	}
	return sessionMutationGateway{mutation: mutationDeps}
}

// NewInviteReadGateway builds the invite read adapter from explicit
// dependencies.
func NewInviteReadGateway(readDeps InviteReadDeps) campaignapp.CampaignInviteReadGateway {
	if readDeps.Invite == nil || readDeps.Participant == nil || readDeps.Social == nil || readDeps.Auth == nil {
		return nil
	}
	return inviteReadGateway{read: readDeps}
}

// NewInviteMutationGateway builds the invite mutation adapter from explicit
// dependencies.
func NewInviteMutationGateway(mutationDeps InviteMutationDeps) campaignapp.CampaignInviteMutationGateway {
	if mutationDeps.Invite == nil || mutationDeps.Auth == nil {
		return nil
	}
	return inviteMutationGateway{mutation: mutationDeps}
}

// NewConfigurationMutationGateway builds the configuration mutation adapter
// from explicit dependencies.
func NewConfigurationMutationGateway(mutationDeps ConfigurationMutationDeps) campaignapp.CampaignConfigurationMutationGateway {
	if mutationDeps.Campaign == nil {
		return nil
	}
	return configurationMutationGateway{mutation: mutationDeps}
}

// NewAutomationReadGateway builds the automation read adapter from explicit
// dependencies.
func NewAutomationReadGateway(readDeps AutomationReadDeps) campaignapp.CampaignAutomationReadGateway {
	if readDeps.Agent == nil {
		return nil
	}
	return automationReadGateway{read: readDeps}
}

// NewAutomationMutationGateway builds the automation mutation adapter from
// explicit dependencies.
func NewAutomationMutationGateway(mutationDeps AutomationMutationDeps) campaignapp.CampaignAutomationMutationGateway {
	if mutationDeps.Campaign == nil {
		return nil
	}
	return automationMutationGateway{mutation: mutationDeps}
}

// NewAuthorizationGateway builds the unary authorization adapter from explicit
// dependencies.
func NewAuthorizationGateway(deps AuthorizationDeps) campaignapp.AuthorizationGateway {
	if deps.Client == nil {
		return nil
	}
	return authorizationGateway{authorization: deps}
}

// NewBatchAuthorizationGateway builds the batch authorization adapter from
// explicit dependencies.
func NewBatchAuthorizationGateway(deps AuthorizationDeps) campaignapp.BatchAuthorizationGateway {
	if deps.Client == nil {
		return nil
	}
	return batchAuthorizationGateway{authorization: deps}
}

// NewCharacterCreationReadGateway builds the character-creation read adapter
// from explicit dependencies.
func NewCharacterCreationReadGateway(deps CharacterCreationReadDeps, assetBaseURL string) campaignapp.CharacterCreationReadGateway {
	if deps.Character == nil || deps.DaggerheartContent == nil || deps.DaggerheartAsset == nil {
		return nil
	}
	return characterCreationReadGateway{read: deps, assetBaseURL: assetBaseURL}
}

// NewCharacterCreationMutationGateway builds the character-creation mutation
// adapter from explicit dependencies.
func NewCharacterCreationMutationGateway(deps CharacterCreationMutationDeps) campaignapp.CharacterCreationMutationGateway {
	if deps.Character == nil {
		return nil
	}
	return characterCreationMutationGateway{mutation: deps}
}
