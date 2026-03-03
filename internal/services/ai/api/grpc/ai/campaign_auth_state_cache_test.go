package ai

import (
	"testing"
	"time"
)

func TestCampaignAIAuthStateCacheGetExpiresEntries(t *testing.T) {
	now := time.Date(2026, 3, 2, 6, 0, 0, 0, time.UTC)
	cache := &campaignAIAuthStateCache{
		byCampaign: map[string]campaignAIAuthState{},
		maxEntries: 10,
		ttl:        time.Minute,
		now:        func() time.Time { return now },
	}
	cache.put(campaignAIAuthState{
		CampaignID:  "camp-1",
		RefreshedAt: now.Add(-2 * time.Minute),
	})

	if _, ok := cache.get("camp-1"); ok {
		t.Fatal("expected expired cache entry to be treated as miss")
	}
	if _, stillPresent := cache.byCampaign["camp-1"]; stillPresent {
		t.Fatal("expected expired cache entry to be evicted")
	}
}

func TestCampaignAIAuthStateCachePutEvictsOldestWhenFull(t *testing.T) {
	now := time.Date(2026, 3, 2, 6, 0, 0, 0, time.UTC)
	cache := &campaignAIAuthStateCache{
		byCampaign: map[string]campaignAIAuthState{},
		maxEntries: 2,
		ttl:        0,
		now:        func() time.Time { return now },
	}
	cache.put(campaignAIAuthState{CampaignID: "camp-1", RefreshedAt: now.Add(-2 * time.Minute)})
	cache.put(campaignAIAuthState{CampaignID: "camp-2", RefreshedAt: now.Add(-time.Minute)})
	cache.put(campaignAIAuthState{CampaignID: "camp-3", RefreshedAt: now})

	if len(cache.byCampaign) != 2 {
		t.Fatalf("cache entries = %d, want 2", len(cache.byCampaign))
	}
	if _, ok := cache.byCampaign["camp-1"]; ok {
		t.Fatal("expected oldest entry to be evicted")
	}
	if _, ok := cache.byCampaign["camp-2"]; !ok {
		t.Fatal("expected camp-2 to remain in cache")
	}
	if _, ok := cache.byCampaign["camp-3"]; !ok {
		t.Fatal("expected camp-3 to be inserted")
	}
}
