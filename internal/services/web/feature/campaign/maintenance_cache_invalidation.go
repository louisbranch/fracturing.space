package campaign

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
)

const DefaultCampaignInvalidationInterval = 30 * time.Second

const (
	// CacheInvalidationIntervalEnv controls how often campaign cache cursors are resynced.
	CacheInvalidationIntervalEnv = "FRACTURING_SPACE_WEB_CACHE_INVALIDATION_INTERVAL"
	// CacheInvalidationMaxCampaignsPerSyncEnv controls the number of campaigns synced per tick.
	CacheInvalidationMaxCampaignsPerSyncEnv = "FRACTURING_SPACE_WEB_CACHE_INVALIDATION_MAX_CAMPAIGNS_PER_SYNC"
)

// Campaign scope constants are shared by cache invalidation behavior.
const (
	CampaignScopeCampaignSummary      = "campaign.summary"
	CampaignScopeCampaignParticipants = "campaign.participants"
	CampaignScopeCampaignSessions     = "campaign.sessions"
	CampaignScopeCampaignCharacters   = "campaign.characters"
	CampaignScopeCampaignInvites      = "campaign.invites"
)

// CampaignScopeRule maps an event type prefix/exact match to stale cache scopes.
type CampaignScopeRule struct {
	prefix string
	exact  string
	scopes []string
}

// CampaignInvalidationScopes is the fallback scope set for stale cache invalidation.
var CampaignInvalidationScopes = []string{
	CampaignScopeCampaignSummary,
	CampaignScopeCampaignParticipants,
	CampaignScopeCampaignSessions,
	CampaignScopeCampaignCharacters,
	CampaignScopeCampaignInvites,
}

// CampaignScopeRules matches event routing to required cache invalidation scopes.
var CampaignScopeRules = []CampaignScopeRule{
	{prefix: "campaign.", scopes: []string{CampaignScopeCampaignSummary}},
	{prefix: "participant.", scopes: []string{CampaignScopeCampaignParticipants, CampaignScopeCampaignSummary}},
	{exact: "seat.reassigned", scopes: []string{CampaignScopeCampaignParticipants, CampaignScopeCampaignSummary}},
	{prefix: "session.", scopes: []string{CampaignScopeCampaignSessions}},
	{prefix: "character.", scopes: []string{CampaignScopeCampaignCharacters, CampaignScopeCampaignSummary}},
	{prefix: "invite.", scopes: []string{CampaignScopeCampaignInvites}},
}

var campaignSyncRoundRobinState struct {
	mu        sync.Mutex
	nextIndex int
}

type invalidationLoopInput struct {
	ctx      context.Context
	interval time.Duration
}

// StartCacheInvalidationWorker starts an async loop that periodically syncs campaign
// cache cursors and marks stale campaign scopes.
func StartCacheInvalidationWorker(store webstorage.Store, eventClient statev1.EventServiceClient) (context.CancelFunc, chan struct{}) {
	if store == nil || eventClient == nil {
		return nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	interval := CacheInvalidationIntervalFromEnv(DefaultCampaignInvalidationInterval)
	maxCampaigns := CacheInvalidationMaxCampaignsPerSyncFromEnv()

	go func() {
		defer close(done)
		RunCacheInvalidationLoop(ctx, store, eventClient, interval, maxCampaigns)
	}()

	return cancel, done
}

// RunCacheInvalidationLoop performs periodic invalidation ticks until context is done.
func RunCacheInvalidationLoop(ctx context.Context, store webstorage.Store, eventClient statev1.EventServiceClient, interval time.Duration, maxCampaigns int) {
	normalized, ok := NormalizeInvalidationLoopInput(ctx, interval, store, eventClient)
	if !ok {
		return
	}

	if err := SyncCampaignEventHeadsWithLimit(normalized.ctx, store, eventClient, maxCampaigns); err != nil {
		log.Printf("cache invalidation sync failed: %v", err)
	}

	ticker := time.NewTicker(normalized.interval)
	defer ticker.Stop()

	for {
		select {
		case <-normalized.ctx.Done():
			return
		case <-ticker.C:
			if err := SyncCampaignEventHeadsWithLimit(normalized.ctx, store, eventClient, maxCampaigns); err != nil {
				log.Printf("cache invalidation sync failed: %v", err)
			}
		}
	}
}

// NormalizeInvalidationLoopInput resolves a usable loop config for cache invalidation.
func NormalizeInvalidationLoopInput(ctx context.Context, interval time.Duration, store webstorage.Store, eventClient statev1.EventServiceClient) (invalidationLoopInput, bool) {
	if store == nil || eventClient == nil {
		return invalidationLoopInput{}, false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if interval <= 0 {
		interval = CacheInvalidationIntervalFromEnv(DefaultCampaignInvalidationInterval)
	}
	return invalidationLoopInput{
		ctx:      ctx,
		interval: interval,
	}, true
}

// SyncCampaignEventHeads wraps SyncCampaignEventHeadsWithLimit with unlimited sync cap.
func SyncCampaignEventHeads(ctx context.Context, store webstorage.Store, eventClient statev1.EventServiceClient) error {
	return SyncCampaignEventHeadsWithLimit(ctx, store, eventClient, 0)
}

// SyncCampaignEventHeadsWithLimit marks stale campaign scopes for tracked campaigns.
func SyncCampaignEventHeadsWithLimit(ctx context.Context, store webstorage.Store, eventClient statev1.EventServiceClient, maxCampaigns int) error {
	if store == nil || eventClient == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	campaignIDs, err := store.ListTrackedCampaignIDs(ctx)
	if err != nil {
		return fmt.Errorf("list tracked campaigns: %w", err)
	}
	campaignIDs = LimitCampaignIDsForSync(campaignIDs, maxCampaigns)
	for _, campaignID := range campaignIDs {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			continue
		}

		headSeq, err := CampaignEventHeadSeq(ctx, eventClient, campaignID)
		if err != nil {
			return fmt.Errorf("read campaign head %q: %w", campaignID, err)
		}
		cursor, ok, err := store.GetCampaignEventCursor(ctx, campaignID)
		if err != nil {
			return fmt.Errorf("read campaign cursor %q: %w", campaignID, err)
		}
		checkedAt := time.Now().UTC()
		deltaScopes := []string(nil)
		if ok && headSeq > cursor.LatestSeq {
			scopes, err := CampaignInvalidationScopesSince(ctx, eventClient, campaignID, cursor.LatestSeq)
			if err != nil {
				return fmt.Errorf("list campaign events for stale scopes %q: %w", campaignID, err)
			}
			deltaScopes = scopes
		}
		for _, scope := range ResolveCampaignStaleScopes(ok, cursor.LatestSeq, headSeq, deltaScopes) {
			if err := store.MarkCampaignScopeStale(ctx, campaignID, scope, headSeq, checkedAt); err != nil {
				return fmt.Errorf("mark stale campaign scope %q for campaign %q: %w", scope, campaignID, err)
			}
		}
		if err := store.PutCampaignEventCursor(ctx, webstorage.CampaignEventCursor{
			CampaignID: campaignID,
			LatestSeq:  headSeq,
			CheckedAt:  checkedAt,
		}); err != nil {
			return fmt.Errorf("persist campaign cursor %q: %w", campaignID, err)
		}
	}
	return nil
}

// LimitCampaignIDsForSync applies round-robin campaign selection.
func LimitCampaignIDsForSync(campaignIDs []string, maxCampaigns int) []string {
	if maxCampaigns <= 0 || len(campaignIDs) <= maxCampaigns {
		return append([]string(nil), campaignIDs...)
	}

	campaignSyncRoundRobinState.mu.Lock()
	defer campaignSyncRoundRobinState.mu.Unlock()

	start := campaignSyncRoundRobinState.nextIndex
	if start < 0 {
		start = 0
	}
	start = start % len(campaignIDs)

	limited := make([]string, 0, maxCampaigns)
	for i := 0; i < maxCampaigns; i++ {
		index := (start + i) % len(campaignIDs)
		limited = append(limited, campaignIDs[index])
	}
	campaignSyncRoundRobinState.nextIndex = (start + maxCampaigns) % len(campaignIDs)
	return limited
}

// CacheInvalidationIntervalFromEnv returns cache invalidation frequency from env.
func CacheInvalidationIntervalFromEnv(defaultInterval time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(CacheInvalidationIntervalEnv))
	if value == "" {
		return defaultInterval
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return defaultInterval
	}
	return parsed
}

// CacheInvalidationMaxCampaignsPerSyncFromEnv reads campaign sync cap from env.
func CacheInvalidationMaxCampaignsPerSyncFromEnv() int {
	value := strings.TrimSpace(os.Getenv(CacheInvalidationMaxCampaignsPerSyncEnv))
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

// CampaignEventHeadSeq reads latest event sequence for campaign from event service.
func CampaignEventHeadSeq(ctx context.Context, eventClient statev1.EventServiceClient, campaignID string) (uint64, error) {
	if eventClient == nil {
		return 0, nil
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
		return 0, fmt.Errorf("list events: %w", err)
	}
	events := resp.GetEvents()
	if len(events) == 0 || events[0] == nil {
		return 0, nil
	}
	return events[0].GetSeq(), nil
}

// CampaignInvalidationScopesSince inspects events and resolves stale scopes.
func CampaignInvalidationScopesSince(ctx context.Context, eventClient statev1.EventServiceClient, campaignID string, afterSeq uint64) ([]string, error) {
	if eventClient == nil {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	scopes := make(map[string]struct{})
	pageToken := ""
	for {
		resp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   200,
			PageToken:  pageToken,
			OrderBy:    "seq",
			AfterSeq:   afterSeq,
		})
		if err != nil {
			return nil, fmt.Errorf("list events: %w", err)
		}
		for _, evt := range resp.GetEvents() {
			if evt == nil {
				continue
			}
			for _, scope := range CampaignScopesForEventType(evt.GetType()) {
				scopes[scope] = struct{}{}
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}

	return sortedScopeKeys(scopes), nil
}

// CampaignScopesForEventType resolves cache scopes for one event type.
func CampaignScopesForEventType(eventType string) []string {
	eventType = strings.TrimSpace(eventType)
	for _, rule := range CampaignScopeRules {
		if rule.exact != "" && eventType == rule.exact {
			return rule.scopes
		}
		if rule.prefix != "" && strings.HasPrefix(eventType, rule.prefix) {
			return rule.scopes
		}
	}
	return DefaultCampaignInvalidationScopes()
}

func sortedScopeKeys(scopeSet map[string]struct{}) []string {
	if len(scopeSet) == 0 {
		return nil
	}
	scopes := make([]string, 0, len(scopeSet))
	for scope := range scopeSet {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	return scopes
}

// DefaultCampaignInvalidationScopes returns all cache scopes.
func DefaultCampaignInvalidationScopes() []string {
	return append([]string(nil), CampaignInvalidationScopes...)
}

// ResolveCampaignStaleScopes decides which stale scopes to mark for a campaign.
func ResolveCampaignStaleScopes(cursorKnown bool, latestSeq, headSeq uint64, deltaScopes []string) []string {
	if !cursorKnown || headSeq <= latestSeq {
		return nil
	}
	if len(deltaScopes) == 0 {
		return DefaultCampaignInvalidationScopes()
	}
	return append([]string(nil), deltaScopes...)
}

// ResetCampaignSyncRoundRobinState resets internal rotation state (test helper).
func ResetCampaignSyncRoundRobinState() {
	campaignSyncRoundRobinState.mu.Lock()
	campaignSyncRoundRobinState.nextIndex = 0
	campaignSyncRoundRobinState.mu.Unlock()
}
