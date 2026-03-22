package routepath

const (
	AppCampaignSessionsPattern      = CampaignsPrefix + "{campaignID}/sessions"
	AppCampaignSessionCreatePattern = CampaignsPrefix + "{campaignID}/sessions/create"
	AppCampaignSessionPattern       = CampaignsPrefix + "{campaignID}/sessions/{sessionID}"
	AppCampaignSessionEndPattern    = CampaignsPrefix + "{campaignID}/sessions/end"
)

// AppCampaignSessions returns the campaign sessions route.
func AppCampaignSessions(campaignID string) string {
	return AppCampaign(campaignID) + "/sessions"
}

// AppCampaignSessionCreate returns the campaign session-create route.
func AppCampaignSessionCreate(campaignID string) string {
	return AppCampaignSessions(campaignID) + "/create"
}

// AppCampaignSessionEnd returns the campaign session-end route.
func AppCampaignSessionEnd(campaignID string) string {
	return AppCampaignSessions(campaignID) + "/end"
}

// AppCampaignSession returns the campaign session-detail route.
func AppCampaignSession(campaignID string, sessionID string) string {
	return AppCampaignSessions(campaignID) + "/" + escapeSegment(sessionID)
}
