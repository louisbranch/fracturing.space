package app

import (
	"context"
	"time"

	"golang.org/x/text/language"
)

const DegradedDependencySocialProfile = "social.profile"
const DegradedDependencyGameCampaigns = "game.campaigns"
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
	DataStatus              DashboardDataStatus
	DegradedDependencies    []string
	ShowPendingProfileBlock bool
	ShowAdventureBlock      bool
	CampaignStartNudges     []CampaignStartNudgeItem
	CampaignStartNudgesMore bool
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

// CampaignStartNudgeActionKind identifies one dashboard CTA mapping.
type CampaignStartNudgeActionKind string

const (
	// CampaignStartNudgeActionKindUnspecified indicates no stable CTA exists.
	CampaignStartNudgeActionKindUnspecified CampaignStartNudgeActionKind = ""
	// CampaignStartNudgeActionKindCreateCharacter asks the viewer to create a character.
	CampaignStartNudgeActionKindCreateCharacter CampaignStartNudgeActionKind = "create_character"
	// CampaignStartNudgeActionKindCompleteCharacter asks the viewer to finish a character.
	CampaignStartNudgeActionKindCompleteCharacter CampaignStartNudgeActionKind = "complete_character"
	// CampaignStartNudgeActionKindConfigureAIAgent asks the viewer to bind an AI agent.
	CampaignStartNudgeActionKindConfigureAIAgent CampaignStartNudgeActionKind = "configure_ai_agent"
	// CampaignStartNudgeActionKindInvitePlayer asks the viewer to invite another player.
	CampaignStartNudgeActionKindInvitePlayer CampaignStartNudgeActionKind = "invite_player"
	// CampaignStartNudgeActionKindManageParticipants asks the viewer to manage participant seats.
	CampaignStartNudgeActionKindManageParticipants CampaignStartNudgeActionKind = "manage_participants"
)

// CampaignStartNudgeItem represents one campaign waiting on the current user.
type CampaignStartNudgeItem struct {
	CampaignID          string
	CampaignName        string
	BlockerCode         string
	BlockerMessage      string
	ActionKind          CampaignStartNudgeActionKind
	TargetParticipantID string
	TargetCharacterID   string
}

// DashboardSnapshot contains userhub dashboard fields used by web rendering logic.
type DashboardSnapshot struct {
	NeedsProfileCompletion       bool
	HasDraftOrActiveCampaign     bool
	CampaignsHasMore             bool
	CampaignStartNudgesAvailable bool
	CampaignStartNudges          []CampaignStartNudgeItem
	CampaignStartNudgesHasMore   bool
	ActiveSessionsAvailable      bool
	ActiveSessions               []ActiveSessionItem
	DegradedDependencies         []string
	Freshness                    DashboardFreshness
	CacheHit                     bool
	GeneratedAt                  time.Time
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
