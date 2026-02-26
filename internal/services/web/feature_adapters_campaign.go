package web

import (
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
)

func (h *handler) campaignFeatureDependenciesImpl() campaignfeature.AppCampaignDependencies {
	var campaignCache *campaignfeature.CampaignCache
	if h != nil {
		campaignCache = campaignfeature.NewCampaignCache(h.cacheStore)
	} else {
		campaignCache = campaignfeature.NewCampaignCache(nil)
	}

	dependencies := campaignfeature.AppCampaignDependencies{}
	buildCampaignFeatureCoreDependencies(h, &dependencies)
	buildCampaignFeatureSessionContextDependencies(h, &dependencies)
	buildCampaignFeatureClientDependencies(h, &dependencies)
	buildCampaignFeatureCacheDependencies(h, campaignCache, &dependencies)
	buildCampaignFeatureInviteDependencies(h, campaignCache, &dependencies)
	buildCampaignFeatureRenderDependencies(h, &dependencies)

	return dependencies
}
