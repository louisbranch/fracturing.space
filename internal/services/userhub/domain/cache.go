package domain

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// dashboardCacheKey identifies one cached dashboard snapshot.
type dashboardCacheKey struct {
	UserID string
	Locale string
}

// dashboardCacheEntry stores one cached dashboard snapshot plus its campaign dependencies.
type dashboardCacheEntry struct {
	Dashboard   Dashboard
	CachedAt    time.Time
	CampaignIDs []string
}

// dashboardCache provides in-memory per-user dashboard snapshots plus reverse
// indexes for targeted invalidation.
type dashboardCache struct {
	mu       sync.RWMutex
	freshTTL time.Duration
	staleTTL time.Duration

	entries      map[dashboardCacheKey]dashboardCacheEntry
	userKeys     map[string]map[dashboardCacheKey]struct{}
	campaignKeys map[string]map[dashboardCacheKey]struct{}
	observer     CampaignDependencyObserver
}

// newDashboardCache creates an in-memory cache with normalized TTLs.
func newDashboardCache(freshTTL, staleTTL time.Duration, observer CampaignDependencyObserver) *dashboardCache {
	if freshTTL <= 0 {
		freshTTL = defaultCacheFreshTTL
	}
	if staleTTL <= 0 {
		staleTTL = defaultCacheStaleTTL
	}
	if staleTTL < freshTTL {
		staleTTL = freshTTL
	}
	return &dashboardCache{
		freshTTL:     freshTTL,
		staleTTL:     staleTTL,
		entries:      make(map[dashboardCacheKey]dashboardCacheEntry),
		userKeys:     make(map[string]map[dashboardCacheKey]struct{}),
		campaignKeys: make(map[string]map[dashboardCacheKey]struct{}),
		observer:     observer,
	}
}

// getFresh returns a dashboard when cache age is within fresh TTL.
func (c *dashboardCache) getFresh(key dashboardCacheKey, now time.Time) (Dashboard, bool) {
	if c == nil {
		return Dashboard{}, false
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return Dashboard{}, false
	}
	age := now.Sub(entry.CachedAt)
	if age < 0 || age > c.freshTTL {
		return Dashboard{}, false
	}
	return cloneDashboard(entry.Dashboard), true
}

// getStale returns a dashboard when cache age is within stale TTL.
func (c *dashboardCache) getStale(key dashboardCacheKey, now time.Time) (Dashboard, bool) {
	if c == nil {
		return Dashboard{}, false
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return Dashboard{}, false
	}
	age := now.Sub(entry.CachedAt)
	if age >= 0 && age <= c.staleTTL {
		return cloneDashboard(entry.Dashboard), true
	}
	if age < 0 {
		return Dashboard{}, false
	}
	c.deleteKeys([]dashboardCacheKey{key})
	return Dashboard{}, false
}

// set upserts one cached dashboard snapshot.
func (c *dashboardCache) set(key dashboardCacheKey, dashboard Dashboard, now time.Time) {
	if c == nil {
		return
	}
	entry := dashboardCacheEntry{
		Dashboard:   cloneDashboard(dashboard),
		CachedAt:    now,
		CampaignIDs: dashboardCampaignDependencies(dashboard),
	}
	var retain, release []string
	c.mu.Lock()
	release = c.removeEntryLocked(key)
	retain = c.addEntryLocked(key, entry)
	c.mu.Unlock()
	c.notifyRetain(retain)
	c.notifyRelease(release)
}

// invalidate removes cache entries tied to the provided users and campaigns.
func (c *dashboardCache) invalidate(userIDs []string, campaignIDs []string) int {
	if c == nil {
		return 0
	}
	keys := make(map[dashboardCacheKey]struct{})

	c.mu.RLock()
	for _, userID := range userIDs {
		for key := range c.userKeys[userID] {
			keys[key] = struct{}{}
		}
	}
	for _, campaignID := range campaignIDs {
		for key := range c.campaignKeys[campaignID] {
			keys[key] = struct{}{}
		}
	}
	c.mu.RUnlock()

	if len(keys) == 0 {
		return 0
	}
	targets := make([]dashboardCacheKey, 0, len(keys))
	for key := range keys {
		targets = append(targets, key)
	}
	return c.deleteKeys(targets)
}

func (c *dashboardCache) deleteKeys(keys []dashboardCacheKey) int {
	if c == nil || len(keys) == 0 {
		return 0
	}
	var release []string
	removed := 0
	c.mu.Lock()
	for _, key := range keys {
		if _, ok := c.entries[key]; !ok {
			continue
		}
		release = append(release, c.removeEntryLocked(key)...)
		removed++
	}
	c.mu.Unlock()
	c.notifyRelease(release)
	return removed
}

func (c *dashboardCache) addEntryLocked(key dashboardCacheKey, entry dashboardCacheEntry) []string {
	c.entries[key] = entry
	if _, ok := c.userKeys[key.UserID]; !ok {
		c.userKeys[key.UserID] = make(map[dashboardCacheKey]struct{})
	}
	c.userKeys[key.UserID][key] = struct{}{}

	retain := make([]string, 0, len(entry.CampaignIDs))
	for _, campaignID := range entry.CampaignIDs {
		keySet, ok := c.campaignKeys[campaignID]
		if !ok {
			keySet = make(map[dashboardCacheKey]struct{})
			c.campaignKeys[campaignID] = keySet
			retain = append(retain, campaignID)
		}
		keySet[key] = struct{}{}
	}
	return retain
}

func (c *dashboardCache) removeEntryLocked(key dashboardCacheKey) []string {
	entry, ok := c.entries[key]
	if !ok {
		return nil
	}
	delete(c.entries, key)

	if userKeySet, ok := c.userKeys[key.UserID]; ok {
		delete(userKeySet, key)
		if len(userKeySet) == 0 {
			delete(c.userKeys, key.UserID)
		}
	}

	release := make([]string, 0, len(entry.CampaignIDs))
	for _, campaignID := range entry.CampaignIDs {
		keySet, ok := c.campaignKeys[campaignID]
		if !ok {
			continue
		}
		delete(keySet, key)
		if len(keySet) == 0 {
			delete(c.campaignKeys, campaignID)
			release = append(release, campaignID)
		}
	}
	return release
}

func (c *dashboardCache) notifyRetain(campaignIDs []string) {
	if c == nil || c.observer == nil {
		return
	}
	for _, campaignID := range normalizeIDs(campaignIDs) {
		c.observer.RetainCampaignDependency(campaignID)
	}
}

func (c *dashboardCache) notifyRelease(campaignIDs []string) {
	if c == nil || c.observer == nil {
		return
	}
	for _, campaignID := range normalizeIDs(campaignIDs) {
		c.observer.ReleaseCampaignDependency(campaignID)
	}
}

func dashboardCampaignDependencies(dashboard Dashboard) []string {
	set := make(map[string]struct{}, len(dashboard.Campaigns.Campaigns)+len(dashboard.Invites.Pending)+len(dashboard.ActiveSessions.Sessions))
	for _, campaign := range dashboard.Campaigns.Campaigns {
		campaignID := strings.TrimSpace(campaign.CampaignID)
		if campaignID == "" {
			continue
		}
		set[campaignID] = struct{}{}
	}
	for _, invite := range dashboard.Invites.Pending {
		campaignID := strings.TrimSpace(invite.CampaignID)
		if campaignID == "" {
			continue
		}
		set[campaignID] = struct{}{}
	}
	for _, activeSession := range dashboard.ActiveSessions.Sessions {
		campaignID := strings.TrimSpace(activeSession.CampaignID)
		if campaignID == "" {
			continue
		}
		set[campaignID] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	campaignIDs := make([]string, 0, len(set))
	for campaignID := range set {
		campaignIDs = append(campaignIDs, campaignID)
	}
	sort.Strings(campaignIDs)
	return campaignIDs
}
