package projection

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"

func registerCampaignProjectionHandlers(r *CoreRouter) {
	HandleProjection(r, campaign.EventTypeCreated, requirements(needsStores(storeCampaign), needsEnvelope(fieldEntityID)), Applier.applyCampaignCreated)
	HandleProjection(r, campaign.EventTypeUpdated, requirements(needsStores(storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyCampaignUpdated)
	HandleProjection(r, campaign.EventTypeAIBound, requirements(needsStores(storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyCampaignAIBound)
	HandleProjection(r, campaign.EventTypeAIUnbound, requirements(needsStores(storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyCampaignAIUnbound)
	HandleProjection(r, campaign.EventTypeAIAuthRotated, requirements(needsStores(storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyCampaignAIAuthRotated)
	HandleProjection(r, campaign.EventTypeForked, requirements(needsStores(storeCampaignFork), needsEnvelope(fieldCampaignID)), Applier.applyCampaignForked)
}
