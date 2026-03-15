package routepath

const (
	AppCampaignAIBindingPattern = CampaignsPrefix + "{campaignID}/ai-binding"
	AppCampaignGamePattern      = CampaignsPrefix + "{campaignID}/game"
)

// AppCampaignAIBinding returns the campaign AI-binding page and submit route.
func AppCampaignAIBinding(campaignID string) string {
	return AppCampaign(campaignID) + "/ai-binding"
}

// AppCampaignGame returns the campaign game route.
func AppCampaignGame(campaignID string) string {
	return AppCampaign(campaignID) + "/game"
}
