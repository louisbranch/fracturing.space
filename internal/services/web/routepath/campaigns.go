package routepath

const (
	AppCampaigns                              = "/app/campaigns"
	AppCampaignsNew                           = "/app/campaigns/new"
	AppCampaignsCreate                        = "/app/campaigns/create"
	CampaignsPrefix                           = "/app/campaigns/"
	AppCampaignPattern                        = CampaignsPrefix + "{campaignID}"
	AppCampaignRestPattern                    = CampaignsPrefix + "{campaignID}/{rest...}"
	AppCampaignEditPattern                    = CampaignsPrefix + "{campaignID}/edit"
	AppCampaignSessionsPattern                = CampaignsPrefix + "{campaignID}/sessions"
	AppCampaignSessionPattern                 = CampaignsPrefix + "{campaignID}/sessions/{sessionID}"
	AppCampaignSessionStartPattern            = CampaignsPrefix + "{campaignID}/sessions/start"
	AppCampaignSessionEndPattern              = CampaignsPrefix + "{campaignID}/sessions/end"
	AppCampaignAIBindingPattern               = CampaignsPrefix + "{campaignID}/ai-binding"
	AppCampaignParticipantsPattern            = CampaignsPrefix + "{campaignID}/participants"
	AppCampaignParticipantCreatePattern       = CampaignsPrefix + "{campaignID}/participants/create"
	AppCampaignParticipantEditPattern         = CampaignsPrefix + "{campaignID}/participants/{participantID}/edit"
	AppCampaignCharactersPattern              = CampaignsPrefix + "{campaignID}/characters"
	AppCampaignCharacterPattern               = CampaignsPrefix + "{campaignID}/characters/{characterID}"
	AppCampaignCharacterEditPattern           = CampaignsPrefix + "{campaignID}/characters/{characterID}/edit"
	AppCampaignCharacterControlPattern        = CampaignsPrefix + "{campaignID}/characters/{characterID}/control"
	AppCampaignCharacterControlClaimPattern   = CampaignsPrefix + "{campaignID}/characters/{characterID}/control/claim"
	AppCampaignCharacterControlReleasePattern = CampaignsPrefix + "{campaignID}/characters/{characterID}/control/release"
	AppCampaignCharacterDeletePattern         = CampaignsPrefix + "{campaignID}/characters/{characterID}/delete"
	AppCampaignCharacterCreatePattern         = CampaignsPrefix + "{campaignID}/characters/create"
	AppCampaignCharacterCreationPattern       = CampaignsPrefix + "{campaignID}/characters/{characterID}/creation"
	AppCampaignCharacterCreationStepPattern   = CampaignsPrefix + "{campaignID}/characters/{characterID}/creation/step"
	AppCampaignCharacterCreationResetPattern  = CampaignsPrefix + "{campaignID}/characters/{characterID}/creation/reset"
	AppCampaignGamePattern                    = CampaignsPrefix + "{campaignID}/game"
	AppCampaignInvitesPattern                 = CampaignsPrefix + "{campaignID}/invites"
	AppCampaignInviteSearchPattern            = CampaignsPrefix + "{campaignID}/invites/search"
	AppCampaignInviteCreatePattern            = CampaignsPrefix + "{campaignID}/invites/create"
	AppCampaignInviteRevokePattern            = CampaignsPrefix + "{campaignID}/invites/revoke"
)

// AppCampaign returns the campaign overview route.
func AppCampaign(campaignID string) string {
	return CampaignsPrefix + escapeSegment(campaignID)
}

// AppCampaignEdit returns the campaign edit route.
func AppCampaignEdit(campaignID string) string {
	return AppCampaign(campaignID) + "/edit"
}

// AppCampaignSessions returns the campaign sessions route.
func AppCampaignSessions(campaignID string) string {
	return AppCampaign(campaignID) + "/sessions"
}

// AppCampaignSessionStart returns the campaign session-start route.
func AppCampaignSessionStart(campaignID string) string {
	return AppCampaignSessions(campaignID) + "/start"
}

// AppCampaignSessionEnd returns the campaign session-end route.
func AppCampaignSessionEnd(campaignID string) string {
	return AppCampaignSessions(campaignID) + "/end"
}

// AppCampaignAIBinding returns the campaign AI-binding page and submit route.
func AppCampaignAIBinding(campaignID string) string {
	return AppCampaign(campaignID) + "/ai-binding"
}

// AppCampaignSession returns the campaign session-detail route.
func AppCampaignSession(campaignID string, sessionID string) string {
	return AppCampaignSessions(campaignID) + "/" + escapeSegment(sessionID)
}

// AppCampaignParticipants returns the campaign participants route.
func AppCampaignParticipants(campaignID string) string {
	return AppCampaign(campaignID) + "/participants"
}

// AppCampaignParticipantCreate returns the campaign participant-create route.
func AppCampaignParticipantCreate(campaignID string) string {
	return AppCampaignParticipants(campaignID) + "/create"
}

// AppCampaignParticipantEdit returns the campaign participant-edit route.
func AppCampaignParticipantEdit(campaignID string, participantID string) string {
	return AppCampaignParticipants(campaignID) + "/" + escapeSegment(participantID) + "/edit"
}

// AppCampaignCharacters returns the campaign characters route.
func AppCampaignCharacters(campaignID string) string {
	return AppCampaign(campaignID) + "/characters"
}

// AppCampaignGame returns the campaign game route.
func AppCampaignGame(campaignID string) string {
	return AppCampaign(campaignID) + "/game"
}

// AppCampaignCharacter returns the campaign character-detail route.
func AppCampaignCharacter(campaignID string, characterID string) string {
	return AppCampaignCharacters(campaignID) + "/" + escapeSegment(characterID)
}

// AppCampaignCharacterEdit returns the campaign character-edit route.
func AppCampaignCharacterEdit(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/edit"
}

// AppCampaignCharacterControl returns the character controller-set route.
func AppCampaignCharacterControl(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/control"
}

// AppCampaignCharacterControlClaim returns the self-claim control route.
func AppCampaignCharacterControlClaim(campaignID string, characterID string) string {
	return AppCampaignCharacterControl(campaignID, characterID) + "/claim"
}

// AppCampaignCharacterControlRelease returns the self-release control route.
func AppCampaignCharacterControlRelease(campaignID string, characterID string) string {
	return AppCampaignCharacterControl(campaignID, characterID) + "/release"
}

// AppCampaignCharacterDelete returns the character-delete route.
func AppCampaignCharacterDelete(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/delete"
}

// AppCampaignCharacterCreation returns the character creation page route.
func AppCampaignCharacterCreation(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/creation"
}

// AppCampaignCharacterCreationStep returns the character creation step route.
func AppCampaignCharacterCreationStep(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/creation/step"
}

// AppCampaignCharacterCreationReset returns the character creation reset route.
func AppCampaignCharacterCreationReset(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/creation/reset"
}

// AppCampaignCharacterCreate returns the campaign character-create route.
func AppCampaignCharacterCreate(campaignID string) string {
	return AppCampaignCharacters(campaignID) + "/create"
}

// AppCampaignInvites returns the campaign invites route.
func AppCampaignInvites(campaignID string) string {
	return AppCampaign(campaignID) + "/invites"
}

// AppCampaignInviteCreate returns the campaign invite-create route.
func AppCampaignInviteCreate(campaignID string) string {
	return AppCampaignInvites(campaignID) + "/create"
}

// AppCampaignInviteSearch returns the invite-search route.
func AppCampaignInviteSearch(campaignID string) string {
	return AppCampaignInvites(campaignID) + "/search"
}

// AppCampaignInviteRevoke returns the campaign invite-revoke route.
func AppCampaignInviteRevoke(campaignID string) string {
	return AppCampaignInvites(campaignID) + "/revoke"
}
