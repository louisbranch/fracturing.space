package ai

import (
	"sync"
	"time"
)

type campaignAIAuthState struct {
	CampaignID      string
	AIAgentID       string
	ActiveSessionID string
	AuthEpoch       uint64
	RefreshedAt     time.Time
}

type campaignAIAuthStateCache struct {
	mu         sync.RWMutex
	byCampaign map[string]campaignAIAuthState
	maxEntries int
	ttl        time.Duration
	now        func() time.Time
}

func newCampaignAIAuthStateCache() *campaignAIAuthStateCache {
	return &campaignAIAuthStateCache{
		byCampaign: make(map[string]campaignAIAuthState),
		maxEntries: 10000,
		ttl:        10 * time.Minute,
		now:        time.Now,
	}
}

func (c *campaignAIAuthStateCache) get(campaignID string) (campaignAIAuthState, bool) {
	if c == nil {
		return campaignAIAuthState{}, false
	}
	c.mu.RLock()
	state, ok := c.byCampaign[campaignID]
	c.mu.RUnlock()
	if !ok {
		return campaignAIAuthState{}, false
	}
	if c.ttl > 0 && !state.RefreshedAt.IsZero() && c.currentTime().UTC().After(state.RefreshedAt.Add(c.ttl)) {
		c.mu.Lock()
		delete(c.byCampaign, campaignID)
		c.mu.Unlock()
		return campaignAIAuthState{}, false
	}
	return state, ok
}

func (c *campaignAIAuthStateCache) put(state campaignAIAuthState) {
	if c == nil {
		return
	}
	if state.CampaignID == "" {
		return
	}
	c.mu.Lock()
	if c.maxEntries > 0 && len(c.byCampaign) >= c.maxEntries {
		c.evictOldestLocked()
	}
	c.byCampaign[state.CampaignID] = state
	c.mu.Unlock()
}

func (c *campaignAIAuthStateCache) currentTime() time.Time {
	if c == nil || c.now == nil {
		return time.Now()
	}
	return c.now()
}

func (c *campaignAIAuthStateCache) evictOldestLocked() {
	var (
		oldestCampaignID string
		oldestRefreshed  time.Time
		first            = true
	)
	for campaignID, state := range c.byCampaign {
		refreshedAt := state.RefreshedAt
		if first || refreshedAt.Before(oldestRefreshed) {
			first = false
			oldestCampaignID = campaignID
			oldestRefreshed = refreshedAt
		}
	}
	if oldestCampaignID != "" {
		delete(c.byCampaign, oldestCampaignID)
	}
}
