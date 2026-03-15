package routepath

const (
	AppCampaignSessionsPattern     = CampaignsPrefix + "{campaignID}/sessions"
	AppCampaignSessionPattern      = CampaignsPrefix + "{campaignID}/sessions/{sessionID}"
	AppCampaignSessionStartPattern = CampaignsPrefix + "{campaignID}/sessions/start"
	AppCampaignSessionEndPattern   = CampaignsPrefix + "{campaignID}/sessions/end"
)

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

// AppCampaignSession returns the campaign session-detail route.
func AppCampaignSession(campaignID string, sessionID string) string {
	return AppCampaignSessions(campaignID) + "/" + escapeSegment(sessionID)
}
