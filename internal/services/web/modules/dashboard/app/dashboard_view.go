package app

import "time"

// DashboardView is the web-dashboard view model derived from userhub state.
type DashboardView struct {
	DataStatus              DashboardDataStatus
	DegradedDependencies    []string
	ShowPendingProfileBlock bool
	PendingInvites          []PendingInviteItem
	ShowAdventureBlock      bool
	CampaignStartNudges     []CampaignStartNudgeItem
	CampaignStartNudgesMore bool
	ActiveSessions          []ActiveSessionItem
	ServiceHealth           []ServiceHealthEntry
}

// PendingInviteItem represents one dashboard link to a pending invite.
type PendingInviteItem struct {
	InviteID        string
	CampaignName    string
	ParticipantName string
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
	InvitesAvailable             bool
	PendingInvites               []PendingInviteItem
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
