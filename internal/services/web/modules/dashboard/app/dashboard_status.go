package app

const DegradedDependencySocialProfile = "social.profile"
const DegradedDependencyGameCampaigns = "game.campaigns"
const DegradedDependencyGameInvites = "game.invites"
const DegradedDependencyGameSessions = "game.sessions"
const DegradedDependencyGameReadiness = "game.readiness"

// DashboardDataStatus describes how complete the rendered dashboard data is.
type DashboardDataStatus string

const (
	// DashboardDataStatusUnspecified indicates the service did not classify the result.
	DashboardDataStatusUnspecified DashboardDataStatus = ""

	// DashboardDataStatusAnonymous indicates no user identity was available for dashboard data.
	DashboardDataStatusAnonymous DashboardDataStatus = "anonymous"

	// DashboardDataStatusReady indicates dashboard data loaded without degraded dependencies.
	DashboardDataStatusReady DashboardDataStatus = "ready"

	// DashboardDataStatusDegraded indicates dashboard data loaded with degraded dependencies.
	DashboardDataStatusDegraded DashboardDataStatus = "degraded"

	// DashboardDataStatusUnavailable indicates dashboard data could not be loaded.
	DashboardDataStatusUnavailable DashboardDataStatus = "unavailable"
)

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
