package web

import (
	"context"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
)

func startCampaignProjectionSubscriptionWorker(store webstorage.Store, eventClient statev1.EventServiceClient) (context.CancelFunc, chan struct{}) {
	return campaignfeature.StartCampaignProjectionSubscriptionWorker(store, eventClient)
}

func runCampaignProjectionSubscriptionLoop(ctx context.Context, store webstorage.Store, eventClient statev1.EventServiceClient) {
	if store == nil || eventClient == nil {
		return
	}
	campaignfeature.RunCampaignProjectionSubscriptionLoop(ctx, store, eventClient)
}

func consumeCampaignProjectionUpdates(ctx context.Context, store webstorage.Store, eventClient statev1.EventServiceClient, campaignID string) {
	campaignfeature.ConsumeCampaignProjectionUpdates(ctx, store, eventClient, campaignID)
}

func campaignProjectionInitialAfterSeq(ctx context.Context, store webstorage.Store, eventClient statev1.EventServiceClient, campaignID string) uint64 {
	return campaignfeature.CampaignProjectionInitialAfterSeq(ctx, store, eventClient, campaignID)
}

func campaignProjectionSubscriptionMaxCampaignsFromEnv() int {
	return campaignfeature.CampaignProjectionSubscriptionMaxCampaignsFromEnv()
}

func selectCampaignIDsForSubscription(campaignIDs []string, maxCampaigns int, nextStart *int) []string {
	return campaignfeature.SelectCampaignIDsForSubscription(campaignIDs, maxCampaigns, nextStart)
}

func waitSubscriptionRetry(ctx context.Context, delay time.Duration) bool {
	return campaignfeature.WaitCampaignSubscriptionRetry(ctx, delay)
}
