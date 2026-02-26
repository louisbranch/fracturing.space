package campaign

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
)

const (
	CampaignProjectionSubscriptionSyncInterval = 30 * time.Second
	campaignProjectionSubscriptionRetryDelay   = time.Second
	DefaultCampaignProjectionMaxCampaigns      = 128
	CampaignProjectionMaxCampaignsEnv          = "FRACTURING_SPACE_WEB_CAMPAIGN_UPDATE_MAX_CAMPAIGNS"
)

// StartCampaignProjectionSubscriptionWorker starts campaign projection subscriptions.
func StartCampaignProjectionSubscriptionWorker(store webstorage.Store, eventClient statev1.EventServiceClient) (context.CancelFunc, chan struct{}) {
	if store == nil || eventClient == nil {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		RunCampaignProjectionSubscriptionLoop(ctx, store, eventClient)
	}()

	return cancel, done
}

// RunCampaignProjectionSubscriptionLoop keeps subscriptions in sync with tracked campaigns.
func RunCampaignProjectionSubscriptionLoop(ctx context.Context, store webstorage.Store, eventClient statev1.EventServiceClient) {
	if ctx == nil {
		ctx = context.Background()
	}
	if store == nil || eventClient == nil {
		return
	}

	var (
		mu          sync.Mutex
		wg          sync.WaitGroup
		subscribers = make(map[string]context.CancelFunc)
		nextStart   int
		maxCampaign = CampaignProjectionSubscriptionMaxCampaignsFromEnv()
	)

	startSubscription := func(campaignID string) {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			return
		}

		mu.Lock()
		if _, exists := subscribers[campaignID]; exists {
			mu.Unlock()
			return
		}
		subCtx, subCancel := context.WithCancel(ctx)
		subscribers[campaignID] = subCancel
		wg.Add(1)
		mu.Unlock()

		go func() {
			defer wg.Done()
			ConsumeCampaignProjectionUpdates(subCtx, store, eventClient, campaignID)
		}()
	}

	syncCampaigns := func() {
		campaignIDs, err := store.ListTrackedCampaignIDs(ctx)
		if err != nil {
			log.Printf("campaign projection subscription sync failed: %v", err)
			return
		}
		selected := SelectCampaignIDsForSubscription(campaignIDs, maxCampaign, &nextStart)
		active := make(map[string]struct{}, len(selected))
		for _, campaignID := range selected {
			campaignID = strings.TrimSpace(campaignID)
			if campaignID == "" {
				continue
			}
			active[campaignID] = struct{}{}
			startSubscription(campaignID)
		}

		mu.Lock()
		for campaignID, cancel := range subscribers {
			if _, ok := active[campaignID]; ok {
				continue
			}
			cancel()
			delete(subscribers, campaignID)
		}
		mu.Unlock()
	}

	syncCampaigns()

	ticker := time.NewTicker(CampaignProjectionSubscriptionSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for _, cancel := range subscribers {
				cancel()
			}
			mu.Unlock()
			wg.Wait()
			return
		case <-ticker.C:
			syncCampaigns()
		}
	}
}

// ConsumeCampaignProjectionUpdates streams campaign updates for one campaign.
func ConsumeCampaignProjectionUpdates(ctx context.Context, store webstorage.Store, eventClient statev1.EventServiceClient, campaignID string) {
	if store == nil || eventClient == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	afterSeq := CampaignProjectionInitialAfterSeq(ctx, store, eventClient, campaignID)

	for {
		if ctx.Err() != nil {
			return
		}

		stream, err := eventClient.SubscribeCampaignUpdates(ctx, &statev1.SubscribeCampaignUpdatesRequest{
			CampaignId: campaignID,
			AfterSeq:   afterSeq,
			Kinds: []statev1.CampaignUpdateKind{
				statev1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED,
			},
		})
		if err != nil {
			if !WaitCampaignSubscriptionRetry(ctx, campaignProjectionSubscriptionRetryDelay) {
				return
			}
			continue
		}

		for {
			update, recvErr := stream.Recv()
			if recvErr != nil {
				break
			}
			// Wiring phase only: advance cursor and intentionally ignore update payload content.
			if update != nil && update.GetSeq() > afterSeq {
				afterSeq = update.GetSeq()
			}
		}

		if !WaitCampaignSubscriptionRetry(ctx, campaignProjectionSubscriptionRetryDelay) {
			return
		}
	}
}

// CampaignProjectionInitialAfterSeq chooses cursor-based projection starting point.
func CampaignProjectionInitialAfterSeq(ctx context.Context, store webstorage.Store, eventClient statev1.EventServiceClient, campaignID string) uint64 {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return 0
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if store != nil {
		cursor, ok, err := store.GetCampaignEventCursor(ctx, campaignID)
		if err == nil && ok {
			return cursor.LatestSeq
		}
	}

	headSeq, err := CampaignEventHeadSeq(ctx, eventClient, campaignID)
	if err != nil {
		return 0
	}
	return headSeq
}

// CampaignProjectionSubscriptionMaxCampaignsFromEnv parses campaign projection subscription cap.
func CampaignProjectionSubscriptionMaxCampaignsFromEnv() int {
	value := strings.TrimSpace(os.Getenv(CampaignProjectionMaxCampaignsEnv))
	if value == "" {
		return DefaultCampaignProjectionMaxCampaigns
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return DefaultCampaignProjectionMaxCampaigns
	}
	return parsed
}

// SelectCampaignIDsForSubscription applies round-robin selection for campaign updates.
func SelectCampaignIDsForSubscription(campaignIDs []string, maxCampaigns int, nextStart *int) []string {
	if maxCampaigns <= 0 || len(campaignIDs) <= maxCampaigns {
		return append([]string(nil), campaignIDs...)
	}

	start := 0
	if nextStart != nil {
		start = *nextStart
	}
	if start < 0 {
		start = 0
	}
	start = start % len(campaignIDs)

	selected := make([]string, 0, maxCampaigns)
	for i := 0; i < maxCampaigns; i++ {
		index := (start + i) % len(campaignIDs)
		selected = append(selected, campaignIDs[index])
	}
	if nextStart != nil {
		*nextStart = (start + maxCampaigns) % len(campaignIDs)
	}
	return selected
}

// WaitCampaignSubscriptionRetry waits before resubscribe with cancellation.
func WaitCampaignSubscriptionRetry(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		delay = campaignProjectionSubscriptionRetryDelay
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
