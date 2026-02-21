package server

import (
	"context"
	"strings"
	"sync"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

const campaignEventSubscriptionRetryDelay = time.Second

type campaignEventCommittedSubscriptionWorker struct {
	ctx         context.Context
	eventClient statev1.EventServiceClient

	mu          sync.Mutex
	subscribers map[string]context.CancelFunc
	wg          sync.WaitGroup
}

func startCampaignEventCommittedSubscriptionWorker(eventClient statev1.EventServiceClient) (func(string), func(string), context.CancelFunc, chan struct{}) {
	if eventClient == nil {
		return nil, nil, nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	worker := &campaignEventCommittedSubscriptionWorker{
		ctx:         ctx,
		eventClient: eventClient,
		subscribers: make(map[string]context.CancelFunc),
	}
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		worker.mu.Lock()
		for _, subCancel := range worker.subscribers {
			subCancel()
		}
		worker.mu.Unlock()
		worker.wg.Wait()
		close(done)
	}()

	return worker.ensureCampaignSubscription, worker.releaseCampaignSubscription, cancel, done
}

func (w *campaignEventCommittedSubscriptionWorker) ensureCampaignSubscription(campaignID string) {
	if w == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	w.mu.Lock()
	if _, exists := w.subscribers[campaignID]; exists {
		w.mu.Unlock()
		return
	}
	subCtx, subCancel := context.WithCancel(w.ctx)
	w.subscribers[campaignID] = subCancel
	w.wg.Add(1)
	w.mu.Unlock()

	go func() {
		defer w.wg.Done()
		consumeCampaignEventCommittedUpdates(subCtx, w.eventClient, campaignID)
	}()
}

func (w *campaignEventCommittedSubscriptionWorker) releaseCampaignSubscription(campaignID string) {
	if w == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	w.mu.Lock()
	cancel, exists := w.subscribers[campaignID]
	if exists {
		delete(w.subscribers, campaignID)
	}
	w.mu.Unlock()
	if exists {
		cancel()
	}
}

func consumeCampaignEventCommittedUpdates(ctx context.Context, eventClient statev1.EventServiceClient, campaignID string) {
	afterSeq := campaignEventCommittedInitialAfterSeq(ctx, eventClient, campaignID)

	for {
		if ctx.Err() != nil {
			return
		}

		stream, err := eventClient.SubscribeCampaignUpdates(ctx, &statev1.SubscribeCampaignUpdatesRequest{
			CampaignId: campaignID,
			AfterSeq:   afterSeq,
			Kinds: []statev1.CampaignUpdateKind{
				statev1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_EVENT_COMMITTED,
			},
		})
		if err != nil {
			if !waitCampaignEventSubscriptionRetry(ctx, campaignEventSubscriptionRetryDelay) {
				return
			}
			continue
		}

		for {
			update, recvErr := stream.Recv()
			if recvErr != nil {
				break
			}
			// Wiring phase only: track cursor and intentionally ignore update payload content.
			if update != nil && update.GetSeq() > afterSeq {
				afterSeq = update.GetSeq()
			}
		}

		if !waitCampaignEventSubscriptionRetry(ctx, campaignEventSubscriptionRetryDelay) {
			return
		}
	}
}

func campaignEventCommittedInitialAfterSeq(ctx context.Context, eventClient statev1.EventServiceClient, campaignID string) uint64 {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" || eventClient == nil {
		return 0
	}
	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
	})
	if err != nil {
		return 0
	}
	events := resp.GetEvents()
	if len(events) == 0 || events[0] == nil {
		return 0
	}
	return events[0].GetSeq()
}

func waitCampaignEventSubscriptionRetry(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		delay = time.Second
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
