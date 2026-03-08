package app

import (
	"context"
	"strings"
	"sync"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

var dashboardProjectionScopes = []string{
	"campaign_summary",
	"campaign_participants",
	"campaign_characters",
	"campaign_sessions",
	"campaign_invites",
}

type campaignDependencyObserver struct {
	retain  func(string)
	release func(string)
}

func (o campaignDependencyObserver) RetainCampaignDependency(campaignID string) {
	if o.retain != nil {
		o.retain(campaignID)
	}
}

func (o campaignDependencyObserver) ReleaseCampaignDependency(campaignID string) {
	if o.release != nil {
		o.release(campaignID)
	}
}

type campaignProjectionSubscriptionManager struct {
	ctx         context.Context
	eventClient gamev1.EventServiceClient
	onUpdate    func(string)

	mu          sync.Mutex
	subscribers map[string]*campaignProjectionSubscriber
	wg          sync.WaitGroup
}

type campaignProjectionSubscriber struct {
	cancel context.CancelFunc
	refs   int
}

func startCampaignProjectionSubscriptionManager(
	ctx context.Context,
	eventClient gamev1.EventServiceClient,
	onUpdate func(string),
) (func(string), func(string), context.CancelFunc, chan struct{}) {
	if ctx == nil || eventClient == nil {
		return nil, nil, nil, nil
	}
	ctx, cancel := context.WithCancel(ctx)
	manager := &campaignProjectionSubscriptionManager{
		ctx:         ctx,
		eventClient: eventClient,
		onUpdate:    onUpdate,
		subscribers: make(map[string]*campaignProjectionSubscriber),
	}
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		manager.mu.Lock()
		for _, subscriber := range manager.subscribers {
			subscriber.cancel()
		}
		manager.mu.Unlock()
		manager.wg.Wait()
		close(done)
	}()
	return manager.retainCampaign, manager.releaseCampaign, cancel, done
}

func (m *campaignProjectionSubscriptionManager) retainCampaign(campaignID string) {
	if m == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	m.mu.Lock()
	if subscriber, ok := m.subscribers[campaignID]; ok {
		subscriber.refs++
		m.mu.Unlock()
		return
	}
	subCtx, subCancel := context.WithCancel(m.ctx)
	m.subscribers[campaignID] = &campaignProjectionSubscriber{
		cancel: subCancel,
		refs:   1,
	}
	m.wg.Add(1)
	m.mu.Unlock()

	go func() {
		defer m.wg.Done()
		consumeCampaignProjectionUpdates(subCtx, m.eventClient, campaignID, m.onUpdate)
	}()
}

func (m *campaignProjectionSubscriptionManager) releaseCampaign(campaignID string) {
	if m == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	m.mu.Lock()
	subscriber, ok := m.subscribers[campaignID]
	if !ok {
		m.mu.Unlock()
		return
	}
	subscriber.refs--
	if subscriber.refs > 0 {
		m.mu.Unlock()
		return
	}
	delete(m.subscribers, campaignID)
	cancel := subscriber.cancel
	m.mu.Unlock()

	cancel()
}

func consumeCampaignProjectionUpdates(ctx context.Context, eventClient gamev1.EventServiceClient, campaignID string, onUpdate func(string)) {
	ctx = campaignProjectionAuthContext(ctx)
	afterSeq := initialCampaignSubscriptionSeq(ctx, eventClient, campaignID)
	for {
		if ctx.Err() != nil {
			return
		}
		stream, err := eventClient.SubscribeCampaignUpdates(ctx, &gamev1.SubscribeCampaignUpdatesRequest{
			CampaignId:       campaignID,
			AfterSeq:         afterSeq,
			Kinds:            []gamev1.CampaignUpdateKind{gamev1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED},
			ProjectionScopes: append([]string{}, dashboardProjectionScopes...),
		})
		if err != nil {
			if !waitCampaignProjectionRetry(ctx) {
				return
			}
			continue
		}
		for {
			update, recvErr := stream.Recv()
			if recvErr != nil {
				break
			}
			if update == nil {
				continue
			}
			if update.GetSeq() > afterSeq {
				afterSeq = update.GetSeq()
			}
			if onUpdate != nil {
				onUpdate(campaignID)
			}
		}
		if !waitCampaignProjectionRetry(ctx) {
			return
		}
	}
}

func initialCampaignSubscriptionSeq(ctx context.Context, eventClient gamev1.EventServiceClient, campaignID string) uint64 {
	if ctx == nil || eventClient == nil {
		return 0
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return 0
	}
	resp, err := eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
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

func campaignProjectionAuthContext(ctx context.Context) context.Context {
	ctx = grpcauthctx.WithServiceID(ctx, "userhub")
	return grpcauthctx.WithAdminOverride(ctx, "userhub dashboard cache projection sync")
}

func waitCampaignProjectionRetry(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
