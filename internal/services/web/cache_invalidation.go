package web

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
)

const cacheInvalidationInterval = 30 * time.Second

var campaignInvalidationScopes = []string{
	cacheScopeCampaignSummary,
	cacheScopeCampaignParticipants,
	cacheScopeCampaignSessions,
	cacheScopeCampaignCharacters,
	cacheScopeCampaignInvites,
}

var campaignScopeRules = []campaignScopeRule{
	{prefix: "campaign.", scopes: []string{cacheScopeCampaignSummary}},
	{prefix: "participant.", scopes: []string{cacheScopeCampaignParticipants, cacheScopeCampaignSummary}},
	{exact: "seat.reassigned", scopes: []string{cacheScopeCampaignParticipants, cacheScopeCampaignSummary}},
	{prefix: "session.", scopes: []string{cacheScopeCampaignSessions}},
	{prefix: "character.", scopes: []string{cacheScopeCampaignCharacters, cacheScopeCampaignSummary}},
	{prefix: "invite.", scopes: []string{cacheScopeCampaignInvites}},
}

type campaignScopeRule struct {
	prefix string
	exact  string
	scopes []string
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
		interval = cacheInvalidationInterval
	}
	return invalidationLoopInput{
		ctx:      ctx,
		interval: interval,
	}, true
}

func startCacheInvalidationWorker(store webstorage.Store, eventClient statev1.EventServiceClient) (context.CancelFunc, chan struct{}) {
	if store == nil || eventClient == nil {
		return nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	h := &handler{
		cacheStore:  store,
		eventClient: eventClient,
	}
	go func() {
		defer close(done)
		h.runCacheInvalidationLoop(ctx, cacheInvalidationInterval)
	}()

	return cancel, done
}

func (h *handler) runCacheInvalidationLoop(ctx context.Context, interval time.Duration) {
	normalized, ok := normalizeInvalidationLoopInput(h, ctx, interval)
	if !ok {
		return
	}

	if err := h.syncCampaignEventHeads(normalized.ctx); err != nil {
		log.Printf("cache invalidation sync failed: %v", err)
	}

	ticker := time.NewTicker(normalized.interval)
	defer ticker.Stop()

	for {
		select {
		case <-normalized.ctx.Done():
			return
		case <-ticker.C:
			if err := h.syncCampaignEventHeads(normalized.ctx); err != nil {
				log.Printf("cache invalidation sync failed: %v", err)
			}
		}
	}
}

func (h *handler) syncCampaignEventHeads(ctx context.Context) error {
	if h == nil || h.cacheStore == nil || h.eventClient == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	campaignIDs, err := h.cacheStore.ListTrackedCampaignIDs(ctx)
	if err != nil {
		return fmt.Errorf("list tracked campaigns: %w", err)
	}
	for _, campaignID := range campaignIDs {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			continue
		}

		headSeq, err := campaignEventHeadSeq(ctx, h.eventClient, campaignID)
		if err != nil {
			return fmt.Errorf("read campaign head %q: %w", campaignID, err)
		}
		cursor, ok, err := h.cacheStore.GetCampaignEventCursor(ctx, campaignID)
		if err != nil {
			return fmt.Errorf("read campaign cursor %q: %w", campaignID, err)
		}
		checkedAt := time.Now().UTC()
		deltaScopes := []string(nil)
		if ok && headSeq > cursor.LatestSeq {
			scopes, err := campaignInvalidationScopesSince(ctx, h.eventClient, campaignID, cursor.LatestSeq)
			if err != nil {
				return fmt.Errorf("list campaign events for stale scopes %q: %w", campaignID, err)
			}
			deltaScopes = scopes
		}
		for _, scope := range resolveCampaignStaleScopes(ok, cursor.LatestSeq, headSeq, deltaScopes) {
			if err := h.cacheStore.MarkCampaignScopeStale(ctx, campaignID, scope, headSeq, checkedAt); err != nil {
				return fmt.Errorf("mark stale campaign scope %q for campaign %q: %w", scope, campaignID, err)
			}
		}
		if err := h.cacheStore.PutCampaignEventCursor(ctx, webstorage.CampaignEventCursor{
			CampaignID: campaignID,
			LatestSeq:  headSeq,
			CheckedAt:  checkedAt,
		}); err != nil {
			return fmt.Errorf("persist campaign cursor %q: %w", campaignID, err)
		}
	}
	return nil
}

func campaignEventHeadSeq(ctx context.Context, eventClient statev1.EventServiceClient, campaignID string) (uint64, error) {
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

func campaignInvalidationScopesSince(ctx context.Context, eventClient statev1.EventServiceClient, campaignID string, afterSeq uint64) ([]string, error) {
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
			for _, scope := range campaignScopesForEventType(evt.GetType()) {
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

func campaignScopesForEventType(eventType string) []string {
	eventType = strings.TrimSpace(eventType)
	for _, rule := range campaignScopeRules {
		if rule.exact != "" && eventType == rule.exact {
			return rule.scopes
		}
		if rule.prefix != "" && strings.HasPrefix(eventType, rule.prefix) {
			return rule.scopes
		}
	}
	return defaultCampaignInvalidationScopes()
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

func defaultCampaignInvalidationScopes() []string {
	return append([]string(nil), campaignInvalidationScopes...)
}

func resolveCampaignStaleScopes(cursorKnown bool, latestSeq, headSeq uint64, deltaScopes []string) []string {
	if !cursorKnown || headSeq <= latestSeq {
		return nil
	}
	if len(deltaScopes) == 0 {
		return defaultCampaignInvalidationScopes()
	}
	return append([]string(nil), deltaScopes...)
}
