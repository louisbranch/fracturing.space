package routepath

const (
	AppCampaigns       = "/app/campaigns"
	AppCampaignsNew    = "/app/campaigns/new"
	AppCampaignsCreate = "/app/campaigns/create"

	CampaignsPrefix        = "/app/campaigns/"
	AppCampaignPattern     = CampaignsPrefix + "{campaignID}"
	AppCampaignRestPattern = CampaignsPrefix + "{campaignID}/{rest...}"
	AppCampaignEditPattern = CampaignsPrefix + "{campaignID}/edit"
)

// AppCampaign returns the campaign overview route.
func AppCampaign(campaignID string) string {
	return CampaignsPrefix + escapeSegment(campaignID)
}

// AppCampaignEdit returns the campaign edit route.
func AppCampaignEdit(campaignID string) string {
	return AppCampaign(campaignID) + "/edit"
}
