package domain

import (
	"testing"
	"time"
)

func TestDashboardCacheInvalidateUserRemovesAllLocaleVariants(t *testing.T) {
	t.Parallel()

	observer := &recordingCampaignDependencyObserver{}
	cache := newDashboardCache(15*time.Second, time.Minute, observer)
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	dashboard := cachedDashboard("camp-1")

	cache.set(dashboardCacheKey{UserID: "user-1", Locale: "en-US"}, dashboard, now)
	cache.set(dashboardCacheKey{UserID: "user-1", Locale: "pt-BR"}, dashboard, now)

	invalidated := cache.invalidate([]string{"user-1"}, nil)
	if invalidated != 2 {
		t.Fatalf("invalidate() = %d, want 2", invalidated)
	}
	if len(cache.entries) != 0 {
		t.Fatalf("entries still present: %v", cache.entries)
	}
	if got := observer.released; len(got) != 1 || got[0] != "camp-1" {
		t.Fatalf("released = %v, want [camp-1]", got)
	}
}

func TestDashboardCacheInvalidateCampaignRemovesPreviewAndInviteDependencies(t *testing.T) {
	t.Parallel()

	observer := &recordingCampaignDependencyObserver{}
	cache := newDashboardCache(15*time.Second, time.Minute, observer)
	now := time.Date(2026, 3, 7, 12, 5, 0, 0, time.UTC)

	cache.set(dashboardCacheKey{UserID: "user-1", Locale: "en-US"}, cachedDashboard("camp-1"), now)
	cache.set(dashboardCacheKey{UserID: "user-2", Locale: "en-US"}, Dashboard{
		Invites: InviteSummary{Pending: []PendingInvite{{CampaignID: "camp-1"}}},
	}, now)

	invalidated := cache.invalidate(nil, []string{"camp-1"})
	if invalidated != 2 {
		t.Fatalf("invalidate() = %d, want 2", invalidated)
	}
	if len(cache.entries) != 0 {
		t.Fatalf("entries still present: %v", cache.entries)
	}
	if got := observer.released; len(got) != 1 || got[0] != "camp-1" {
		t.Fatalf("released = %v, want [camp-1]", got)
	}
}

func TestDashboardCacheGetStaleExpiresEntryAndReleasesCampaignDependency(t *testing.T) {
	t.Parallel()

	observer := &recordingCampaignDependencyObserver{}
	cache := newDashboardCache(5*time.Second, 10*time.Second, observer)
	cachedAt := time.Date(2026, 3, 7, 12, 10, 0, 0, time.UTC)
	key := dashboardCacheKey{UserID: "user-1", Locale: "en-US"}

	cache.set(key, cachedDashboard("camp-9"), cachedAt)
	if _, ok := cache.getStale(key, cachedAt.Add(11*time.Second)); ok {
		t.Fatalf("getStale() ok = true, want false after stale TTL expiry")
	}
	if _, ok := cache.entries[key]; ok {
		t.Fatalf("entry %v still present after expiry", key)
	}
	if got := observer.released; len(got) != 1 || got[0] != "camp-9" {
		t.Fatalf("released = %v, want [camp-9]", got)
	}
}

type recordingCampaignDependencyObserver struct {
	retained []string
	released []string
}

func (o *recordingCampaignDependencyObserver) RetainCampaignDependency(campaignID string) {
	o.retained = append(o.retained, campaignID)
}

func (o *recordingCampaignDependencyObserver) ReleaseCampaignDependency(campaignID string) {
	o.released = append(o.released, campaignID)
}

func cachedDashboard(campaignID string) Dashboard {
	return Dashboard{
		Campaigns: CampaignSummary{
			Campaigns: []CampaignPreview{{CampaignID: campaignID}},
		},
	}
}
