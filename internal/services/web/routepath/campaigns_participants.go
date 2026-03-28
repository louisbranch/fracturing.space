package routepath

const (
	AppCampaignParticipantsPattern      = CampaignsPrefix + "{campaignID}/participants"
	AppCampaignParticipantCreatePattern = CampaignsPrefix + "{campaignID}/participants/create"
	AppCampaignParticipantEditPattern   = CampaignsPrefix + "{campaignID}/participants/{participantID}/edit"
	AppCampaignParticipantDeletePattern = CampaignsPrefix + "{campaignID}/participants/{participantID}/delete"
)

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

// AppCampaignParticipantDelete returns the campaign participant-delete route.
func AppCampaignParticipantDelete(campaignID string, participantID string) string {
	return AppCampaignParticipants(campaignID) + "/" + escapeSegment(participantID) + "/delete"
}
