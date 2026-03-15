package routepath

const (
	AppCampaignStarters             = "/app/campaigns/starters"
	AppCampaignStarterPattern       = AppCampaignStarters + "/preview/{starterKey}"
	AppCampaignStarterLaunchPattern = AppCampaignStarters + "/launch/{starterKey}"
)

// AppCampaignStarter returns the protected starter preview route.
func AppCampaignStarter(starterKey string) string {
	return AppCampaignStarters + "/preview/" + escapeSegment(starterKey)
}

// AppCampaignStarterLaunch returns the protected starter launch route.
func AppCampaignStarterLaunch(starterKey string) string {
	return AppCampaignStarters + "/launch/" + escapeSegment(starterKey)
}
