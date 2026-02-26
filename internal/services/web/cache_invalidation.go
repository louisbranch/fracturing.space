package web

import (
	"context"
	"sort"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
)

const cacheInvalidationInterval = campaignfeature.DefaultCampaignInvalidationInterval

const (
	cacheInvalidationIntervalEnv            = campaignfeature.CacheInvalidationIntervalEnv
	cacheInvalidationMaxCampaignsPerSyncEnv = campaignfeature.CacheInvalidationMaxCampaignsPerSyncEnv
)

const (
	cacheScopeCampaignSummary      = "campaign.summary"
	cacheScopeCampaignParticipants = "campaign.participants"
	cacheScopeCampaignSessions     = "campaign.sessions"
	cacheScopeCampaignCharacters   = "campaign.characters"
	cacheScopeCampaignInvites      = "campaign.invites"
)

var campaignInvalidationScopes = []string{
	cacheScopeCampaignSummary,
	cacheScopeCampaignParticipants,
	cacheScopeCampaignSessions,
	cacheScopeCampaignCharacters,
	cacheScopeCampaignInvites,
}

type invalidationLoopInput struct {
	ctx      context.Context
	interval time.Duration
}

func normalizeInvalidationLoopInput(h *handler, ctx context.Context, interval time.Duration) (invalidationLoopInput, bool) {
	if h == nil {
		return invalidationLoopInput{}, false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if interval <= 0 {
		interval = campaignfeature.CacheInvalidationIntervalFromEnv(cacheInvalidationInterval)
	}
	return invalidationLoopInput{
		ctx:      ctx,
		interval: interval,
	}, true
}

func startCacheInvalidationWorker(store webstorage.Store, eventClient statev1.EventServiceClient) (context.CancelFunc, chan struct{}) {
	return campaignfeature.StartCacheInvalidationWorker(store, eventClient)
}

func (h *handler) runCacheInvalidationLoop(ctx context.Context, interval time.Duration, maxCampaigns int) {
	normalized, ok := normalizeInvalidationLoopInput(h, ctx, interval)
	if !ok {
		return
	}
	campaignfeature.RunCacheInvalidationLoop(normalized.ctx, h.cacheStore, h.eventClient, normalized.interval, maxCampaigns)
}

func (h *handler) syncCampaignEventHeads(ctx context.Context) error {
	return h.syncCampaignEventHeadsWithLimit(ctx, 0)
}

func (h *handler) syncCampaignEventHeadsWithLimit(ctx context.Context, maxCampaigns int) error {
	if h == nil || h.cacheStore == nil || h.eventClient == nil {
		return nil
	}
	return campaignfeature.SyncCampaignEventHeadsWithLimit(ctx, h.cacheStore, h.eventClient, maxCampaigns)
}

func limitCampaignIDsForSync(campaignIDs []string, maxCampaigns int) []string {
	return campaignfeature.LimitCampaignIDsForSync(campaignIDs, maxCampaigns)
}

func resetCampaignSyncRoundRobinState() {
	campaignfeature.ResetCampaignSyncRoundRobinState()
}

func cacheInvalidationIntervalFromEnv(defaultInterval time.Duration) time.Duration {
	return campaignfeature.CacheInvalidationIntervalFromEnv(defaultInterval)
}

func cacheInvalidationMaxCampaignsPerSyncFromEnv() int {
	return campaignfeature.CacheInvalidationMaxCampaignsPerSyncFromEnv()
}

func campaignEventHeadSeq(ctx context.Context, eventClient statev1.EventServiceClient, campaignID string) (uint64, error) {
	return campaignfeature.CampaignEventHeadSeq(ctx, eventClient, campaignID)
}

func campaignInvalidationScopesSince(ctx context.Context, eventClient statev1.EventServiceClient, campaignID string, afterSeq uint64) ([]string, error) {
	return campaignfeature.CampaignInvalidationScopesSince(ctx, eventClient, campaignID, afterSeq)
}

func campaignScopesForEventType(eventType string) []string {
	return campaignfeature.CampaignScopesForEventType(eventType)
}

func resolveCampaignStaleScopes(cursorKnown bool, latestSeq, headSeq uint64, deltaScopes []string) []string {
	return campaignfeature.ResolveCampaignStaleScopes(cursorKnown, latestSeq, headSeq, deltaScopes)
}

func defaultCampaignInvalidationScopes() []string {
	return campaignfeature.DefaultCampaignInvalidationScopes()
}

func sortedScopeKeys(scopeSet map[string]struct{}) []string {
	if len(scopeSet) == 0 {
		return nil
	}
	out := make([]string, 0, len(scopeSet))
	for scope := range scopeSet {
		out = append(out, scope)
	}
	sort.Strings(out)
	return out
}
