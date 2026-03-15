package app

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"golang.org/x/text/language"
	"google.golang.org/grpc/metadata"
)

func contextWithResolvedUserID(userID string) context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(grpcmeta.UserIDHeader, userID))
}

type campaignGatewayStub struct {
	items                                   []CampaignSummary
	listErr                                 error
	campaignName                            string
	campaignNameErr                         error
	campaignWorkspace                       CampaignWorkspace
	campaignWorkspaceErr                    error
	campaignGameSurface                     CampaignGameSurface
	campaignGameSurfaceErr                  error
	campaignAIAgents                        []CampaignAIAgentOption
	campaignAIAgentsErr                     error
	campaignParticipants                    []CampaignParticipant
	campaignParticipantsErr                 error
	campaignParticipant                     CampaignParticipant
	campaignParticipantErr                  error
	campaignCharacters                      []CampaignCharacter
	campaignCharactersErr                   error
	campaignSessions                        []CampaignSession
	campaignSessionsErr                     error
	campaignSessionReadiness                CampaignSessionReadiness
	campaignSessionReadinessErr             error
	campaignInvites                         []CampaignInvite
	campaignInvitesErr                      error
	inviteSearchResults                     []InviteUserSearchResult
	inviteSearchErr                         error
	lastSearchInviteUsersInput              SearchInviteUsersInput
	searchInviteCalls                       int
	createCampaignResult                    CreateCampaignResult
	createCampaignErr                       error
	lastCreateInput                         CreateCampaignInput
	updateCampaignErr                       error
	lastUpdateCampaignInput                 UpdateCampaignInput
	updateCampaignAIBindingErr              error
	lastUpdateCampaignAIBindingInput        UpdateCampaignAIBindingInput
	authorizationDecision                   AuthorizationDecision
	authorizationErr                        error
	authorizationCalls                      int
	authorizationRequests                   []campaignAuthorizationRequest
	batchAuthorizationDecisions             []AuthorizationDecision
	batchAuthorizationErr                   error
	batchAuthorizationCalls                 int
	batchAuthorizationRequests              []AuthorizationCheck
	characterCreationProgress               CampaignCharacterCreationProgress
	characterCreationProgressErr            error
	characterCreationCatalog                CampaignCharacterCreationCatalog
	characterCreationCatalogErr             error
	characterCreationCatalogLocale          language.Tag
	characterCreationProfile                CampaignCharacterCreationProfile
	characterCreationProfileErr             error
	createCharacterResult                   CreateCharacterResult
	createCharacterResultSet                bool
	createCharacterErr                      error
	updateCharacterErr                      error
	deleteCharacterErr                      error
	setCharacterControllerErr               error
	claimCharacterControlErr                error
	releaseCharacterControlErr              error
	lastUpdateCharacterCampaignID           string
	lastUpdateCharacterID                   string
	lastUpdateCharacterInput                UpdateCharacterInput
	lastDeleteCharacterCampaignID           string
	lastDeleteCharacterID                   string
	lastSetCharacterControllerCampaignID    string
	lastSetCharacterControllerCharacterID   string
	lastSetCharacterControllerParticipantID string
	lastClaimCharacterControlCampaignID     string
	lastClaimCharacterControlCharacterID    string
	lastReleaseCharacterControlCampaignID   string
	lastReleaseCharacterControlCharacterID  string
	lastStartSessionInput                   StartSessionInput
	lastEndSessionInput                     EndSessionInput
	lastCreateInviteInput                   CreateInviteInput
	lastCreateParticipantInput              CreateParticipantInput
	lastRevokeInviteInput                   RevokeInviteInput
	lastUpdateParticipantInput              UpdateParticipantInput
	applyCharacterCreationStepErr           error
	resetCharacterCreationWorkflowErr       error
	calls                                   []string
}

type campaignAuthorizationRequest struct {
	Action   AuthorizationAction
	Resource AuthorizationResource
	Target   *AuthorizationTarget
}

func (f *campaignGatewayStub) ListCampaigns(context.Context) ([]CampaignSummary, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.items, nil
}

func (f *campaignGatewayStub) CampaignName(context.Context, string) (string, error) {
	if f.campaignNameErr != nil {
		return "", f.campaignNameErr
	}
	return f.campaignName, nil
}

func (f *campaignGatewayStub) CampaignWorkspace(_ context.Context, campaignID string) (CampaignWorkspace, error) {
	if f.campaignWorkspaceErr != nil {
		return CampaignWorkspace{}, f.campaignWorkspaceErr
	}
	workspace := f.campaignWorkspace
	if strings.TrimSpace(workspace.ID) == "" {
		workspace.ID = campaignID
	}
	return workspace, nil
}

func (f *campaignGatewayStub) CampaignGameSurface(_ context.Context, campaignID string) (CampaignGameSurface, error) {
	if f.campaignGameSurfaceErr != nil {
		return CampaignGameSurface{}, f.campaignGameSurfaceErr
	}
	surface := f.campaignGameSurface
	if strings.TrimSpace(surface.SessionID) == "" {
		surface.SessionID = "sess-1"
	}
	if strings.TrimSpace(surface.SessionName) == "" {
		surface.SessionName = "Session One"
	}
	if strings.TrimSpace(surface.Participant.ID) == "" {
		surface.Participant.ID = "p1"
	}
	if strings.TrimSpace(surface.Participant.Name) == "" {
		surface.Participant.Name = "Owner"
	}
	if strings.TrimSpace(surface.Participant.Role) == "" {
		surface.Participant.Role = "Player"
	}
	if surface.ActiveScene == nil {
		surface.ActiveScene = &CampaignGameScene{
			ID:        "scene-1",
			SessionID: surface.SessionID,
			Name:      "Bridge Watch",
			Characters: []CampaignGameCharacter{
				{ID: "char-1", Name: "Aria", OwnerParticipantID: surface.Participant.ID},
			},
		}
	}
	if surface.PlayerPhase == nil {
		surface.PlayerPhase = &CampaignGamePlayerPhase{
			PhaseID:              "phase-1",
			Status:               "players",
			ActingCharacterIDs:   []string{"char-1"},
			ActingParticipantIDs: []string{surface.Participant.ID},
			Slots:                []CampaignGamePlayerSlot{},
		}
	}
	if len(surface.OOC.Posts) == 0 {
		surface.OOC.Posts = []CampaignGameOOCPost{}
	}
	return surface, nil
}

func (f *campaignGatewayStub) CampaignAIAgents(context.Context) ([]CampaignAIAgentOption, error) {
	if f.campaignAIAgentsErr != nil {
		return nil, f.campaignAIAgentsErr
	}
	return f.campaignAIAgents, nil
}

func (f *campaignGatewayStub) CampaignParticipants(context.Context, string) ([]CampaignParticipant, error) {
	if f.campaignParticipantsErr != nil {
		return nil, f.campaignParticipantsErr
	}
	return f.campaignParticipants, nil
}

func (f *campaignGatewayStub) CampaignParticipant(_ context.Context, _ string, participantID string) (CampaignParticipant, error) {
	if f.campaignParticipantErr != nil {
		return CampaignParticipant{}, f.campaignParticipantErr
	}
	if strings.TrimSpace(f.campaignParticipant.ID) != "" {
		return f.campaignParticipant, nil
	}
	for _, participant := range f.campaignParticipants {
		if strings.TrimSpace(participant.ID) == strings.TrimSpace(participantID) {
			return participant, nil
		}
	}
	return CampaignParticipant{ID: strings.TrimSpace(participantID)}, nil
}

func (f *campaignGatewayStub) CampaignCharacters(context.Context, string, CharacterReadContext) ([]CampaignCharacter, error) {
	if f.campaignCharactersErr != nil {
		return nil, f.campaignCharactersErr
	}
	return f.campaignCharacters, nil
}

func (f *campaignGatewayStub) CampaignCharacter(_ context.Context, _ string, characterID string, _ CharacterReadContext) (CampaignCharacter, error) {
	if f.campaignCharactersErr != nil {
		return CampaignCharacter{}, f.campaignCharactersErr
	}
	for _, character := range f.campaignCharacters {
		if strings.TrimSpace(character.ID) == strings.TrimSpace(characterID) {
			return character, nil
		}
	}
	return CampaignCharacter{ID: strings.TrimSpace(characterID)}, nil
}

func (f *campaignGatewayStub) CampaignSessions(context.Context, string) ([]CampaignSession, error) {
	if f.campaignSessionsErr != nil {
		return nil, f.campaignSessionsErr
	}
	return f.campaignSessions, nil
}

func (f *campaignGatewayStub) CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error) {
	if f.campaignSessionReadinessErr != nil {
		return CampaignSessionReadiness{}, f.campaignSessionReadinessErr
	}
	if !f.campaignSessionReadiness.Ready && len(f.campaignSessionReadiness.Blockers) == 0 {
		return CampaignSessionReadiness{Ready: true, Blockers: []CampaignSessionReadinessBlocker{}}, nil
	}
	return f.campaignSessionReadiness, nil
}

func (f *campaignGatewayStub) CampaignInvites(context.Context, string) ([]CampaignInvite, error) {
	if f.campaignInvitesErr != nil {
		return nil, f.campaignInvitesErr
	}
	return f.campaignInvites, nil
}

func (f *campaignGatewayStub) SearchInviteUsers(_ context.Context, input SearchInviteUsersInput) ([]InviteUserSearchResult, error) {
	f.searchInviteCalls++
	f.lastSearchInviteUsersInput = input
	if f.inviteSearchErr != nil {
		return nil, f.inviteSearchErr
	}
	return f.inviteSearchResults, nil
}

func (f *campaignGatewayStub) CharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error) {
	if f.characterCreationProgressErr != nil {
		return CampaignCharacterCreationProgress{}, f.characterCreationProgressErr
	}
	return f.characterCreationProgress, nil
}

func (f *campaignGatewayStub) CharacterCreationCatalog(_ context.Context, locale language.Tag) (CampaignCharacterCreationCatalog, error) {
	f.characterCreationCatalogLocale = locale
	if f.characterCreationCatalogErr != nil {
		return CampaignCharacterCreationCatalog{}, f.characterCreationCatalogErr
	}
	return f.characterCreationCatalog, nil
}

func (f *campaignGatewayStub) CharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error) {
	if f.characterCreationProfileErr != nil {
		return CampaignCharacterCreationProfile{}, f.characterCreationProfileErr
	}
	return f.characterCreationProfile, nil
}

func (f *campaignGatewayStub) CreateCampaign(_ context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	if f != nil {
		// capture input for behavior assertions
		f.lastCreateInput = input
	}
	if f.createCampaignErr != nil {
		return CreateCampaignResult{}, f.createCampaignErr
	}
	if f.createCampaignResult.CampaignID == "" {
		return CreateCampaignResult{CampaignID: "created"}, nil
	}
	return f.createCampaignResult, nil
}

func (f *campaignGatewayStub) UpdateCampaign(_ context.Context, _ string, input UpdateCampaignInput) error {
	f.lastUpdateCampaignInput = input
	f.calls = append(f.calls, "update-campaign")
	return f.updateCampaignErr
}

func (f *campaignGatewayStub) UpdateCampaignAIBinding(_ context.Context, _ string, input UpdateCampaignAIBindingInput) error {
	f.lastUpdateCampaignAIBindingInput = input
	f.calls = append(f.calls, "update-campaign-ai-binding")
	return f.updateCampaignAIBindingErr
}

func (f *campaignGatewayStub) StartSession(_ context.Context, _ string, input StartSessionInput) error {
	f.lastStartSessionInput = input
	f.calls = append(f.calls, "start")
	return nil
}

func (f *campaignGatewayStub) EndSession(_ context.Context, _ string, input EndSessionInput) error {
	f.lastEndSessionInput = input
	f.calls = append(f.calls, "end")
	return nil
}

func (f *campaignGatewayStub) CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error) {
	f.calls = append(f.calls, "create-character")
	if f.createCharacterErr != nil {
		return CreateCharacterResult{}, f.createCharacterErr
	}
	if !f.createCharacterResultSet {
		return CreateCharacterResult{CharacterID: "char-created"}, nil
	}
	return f.createCharacterResult, nil
}

func (f *campaignGatewayStub) UpdateCharacter(_ context.Context, campaignID string, characterID string, input UpdateCharacterInput) error {
	f.lastUpdateCharacterCampaignID = campaignID
	f.lastUpdateCharacterID = characterID
	f.lastUpdateCharacterInput = input
	f.calls = append(f.calls, "update-character")
	return f.updateCharacterErr
}

func (f *campaignGatewayStub) DeleteCharacter(_ context.Context, campaignID string, characterID string) error {
	f.lastDeleteCharacterCampaignID = campaignID
	f.lastDeleteCharacterID = characterID
	f.calls = append(f.calls, "delete-character")
	return f.deleteCharacterErr
}

func (f *campaignGatewayStub) SetCharacterController(_ context.Context, campaignID string, characterID string, participantID string) error {
	f.lastSetCharacterControllerCampaignID = campaignID
	f.lastSetCharacterControllerCharacterID = characterID
	f.lastSetCharacterControllerParticipantID = participantID
	f.calls = append(f.calls, "set-character-controller")
	return f.setCharacterControllerErr
}

func (f *campaignGatewayStub) ClaimCharacterControl(_ context.Context, campaignID string, characterID string) error {
	f.lastClaimCharacterControlCampaignID = campaignID
	f.lastClaimCharacterControlCharacterID = characterID
	f.calls = append(f.calls, "claim-character-control")
	return f.claimCharacterControlErr
}

func (f *campaignGatewayStub) ReleaseCharacterControl(_ context.Context, campaignID string, characterID string) error {
	f.lastReleaseCharacterControlCampaignID = campaignID
	f.lastReleaseCharacterControlCharacterID = characterID
	f.calls = append(f.calls, "release-character-control")
	return f.releaseCharacterControlErr
}

func (f *campaignGatewayStub) CreateParticipant(_ context.Context, _ string, input CreateParticipantInput) (CreateParticipantResult, error) {
	f.lastCreateParticipantInput = input
	f.calls = append(f.calls, "create-participant")
	return CreateParticipantResult{ParticipantID: "participant-created"}, nil
}

func (f *campaignGatewayStub) UpdateParticipant(_ context.Context, _ string, input UpdateParticipantInput) error {
	f.lastUpdateParticipantInput = input
	f.calls = append(f.calls, "update-participant")
	return nil
}

func (f *campaignGatewayStub) CreateInvite(_ context.Context, _ string, input CreateInviteInput) error {
	f.lastCreateInviteInput = input
	f.calls = append(f.calls, "create-invite")
	return nil
}

func (f *campaignGatewayStub) RevokeInvite(_ context.Context, _ string, input RevokeInviteInput) error {
	f.lastRevokeInviteInput = input
	f.calls = append(f.calls, "revoke-invite")
	return nil
}

func (f *campaignGatewayStub) ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error {
	f.calls = append(f.calls, "apply-character-creation-step")
	return f.applyCharacterCreationStepErr
}

func (f *campaignGatewayStub) ResetCharacterCreationWorkflow(context.Context, string, string) error {
	f.calls = append(f.calls, "reset-character-creation-workflow")
	return f.resetCharacterCreationWorkflowErr
}

func (f *campaignGatewayStub) CanCampaignAction(
	_ context.Context,
	_ string,
	action AuthorizationAction,
	resource AuthorizationResource,
	target *AuthorizationTarget,
) (AuthorizationDecision, error) {
	f.authorizationCalls++
	f.authorizationRequests = append(f.authorizationRequests, campaignAuthorizationRequest{Action: action, Resource: resource, Target: target})
	if f.authorizationErr != nil {
		return AuthorizationDecision{}, f.authorizationErr
	}
	return f.authorizationDecision, nil
}

func (f *campaignGatewayStub) BatchCanCampaignAction(
	_ context.Context,
	_ string,
	checks []AuthorizationCheck,
) ([]AuthorizationDecision, error) {
	f.batchAuthorizationCalls++
	f.batchAuthorizationRequests = append(f.batchAuthorizationRequests, checks...)
	if f.batchAuthorizationErr != nil {
		return nil, f.batchAuthorizationErr
	}
	return f.batchAuthorizationDecisions, nil
}
