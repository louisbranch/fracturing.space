package web

import (
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
)

type appCampaignRouteDependencies struct {
	appCampaignDependencies campaignfeature.AppCampaignDependencies
}

func (h *handler) appCampaignRouteDependencies() appCampaignRouteDependencies {
	return appCampaignRouteDependencies{
		appCampaignDependencies: h.campaignFeatureDependenciesImpl(),
	}
}
