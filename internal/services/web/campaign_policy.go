package web

import statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"

// canManageCampaignAccess reports whether a campaign access level can execute manager actions.
func canManageCampaignAccess(access statev1.CampaignAccess) bool {
	return access == statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER || access == statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
}
