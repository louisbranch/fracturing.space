package app

import (
	"context"
	"time"

	"golang.org/x/text/language"
)

const DegradedDependencySocialProfile = "social.profile"
const DegradedDependencyGameCampaigns = "game.campaigns"
const DegradedDependencyGameSessions = "game.sessions"

// ServiceHealthEntry represents the availability status of a backend service group.
type ServiceHealthEntry struct {
	Label     string
	Available bool
}

// HealthProvider returns current service health entries.
// Called on each dashboard load to get live status.
type HealthProvider func(ctx context.Context) []ServiceHealthEntry

// DashboardView is the web-dashboard view model derived from userhub state.
type DashboardView struct {
	ShowPendingProfileBlock bool
	ShowAdventureBlock      bool
	ActiveSessions          []ActiveSessionItem
	ServiceHealth           []ServiceHealthEntry
}

// ActiveSessionItem represents one dashboard join row for an active campaign session.
type ActiveSessionItem struct {
	CampaignID   string
	CampaignName string
	SessionID    string
	SessionName  string
}

// DashboardSnapshot contains userhub dashboard fields used by web rendering logic.
type DashboardSnapshot struct {
	NeedsProfileCompletion   bool
	HasDraftOrActiveCampaign bool
	CampaignsHasMore         bool
	ActiveSessionsAvailable  bool
	ActiveSessions           []ActiveSessionItem
	DegradedDependencies     []string
	Freshness                DashboardFreshness
	CacheHit                 bool
	GeneratedAt              time.Time
}

// DashboardFreshness preserves userhub freshness metadata for observability.
type DashboardFreshness int

const (
	// DashboardFreshnessUnspecified indicates no freshness metadata was provided.
	DashboardFreshnessUnspecified DashboardFreshness = iota
	// DashboardFreshnessFresh indicates live or fresh-cache data.
	DashboardFreshnessFresh
	// DashboardFreshnessStale indicates stale-cache fallback data.
	DashboardFreshnessStale
)

// Gateway loads dashboard snapshot data for one user.
type Gateway interface {
	LoadDashboard(context.Context, string, language.Tag) (DashboardSnapshot, error)
}

// Service exposes dashboard orchestration methods used by transport handlers.
type Service interface {
	LoadDashboard(context.Context, string, language.Tag) (DashboardView, error)
}
