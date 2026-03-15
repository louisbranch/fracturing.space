package routepath

const (
	AppCampaignInvitesPattern      = CampaignsPrefix + "{campaignID}/invites"
	AppCampaignInviteSearchPattern = CampaignsPrefix + "{campaignID}/invites/search"
	AppCampaignInviteCreatePattern = CampaignsPrefix + "{campaignID}/invites/create"
	AppCampaignInviteRevokePattern = CampaignsPrefix + "{campaignID}/invites/revoke"
)

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
