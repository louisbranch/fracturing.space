package campaigns

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// CampaignChatView carries the game chat page state for one campaign workspace.
type CampaignChatView struct {
	CampaignID           string
	CampaignName         string
	BackURL              string
	BootstrapJSON        string
	ParticipantName      string
	ParticipantRole      string
	SessionName          string
	SceneName            string
	SceneDescription     string
	PhaseStatus          string
	PhaseLabel           string
	PhaseFrame           string
	OOCSummary           string
	HasOpenOOC           bool
	ActingCharacters     []CampaignChatCharacterView
	SceneCharacters      []CampaignChatCharacterView
	Slots                []CampaignChatPlayerSlotView
	OOCPosts             []CampaignChatOOCPostView
	YieldedParticipants  []string
	OOCReadyParticipants []string
	GMAuthorityLabel     string
	AITurnStatus         string
	AITurnSummary        string
	AITurnError          string
}

// CampaignChatCharacterView carries one visible scene character row.
type CampaignChatCharacterView struct {
	CharacterID        string
	Name               string
	OwnerParticipantID string
	Active             bool
}

// CampaignChatPlayerSlotView carries one participant-owned player slot.
type CampaignChatPlayerSlotView struct {
	ParticipantID        string
	SummaryText          string
	CharacterLabel       string
	Yielded              bool
	ReviewStatus         string
	ReviewLabel          string
	ReviewReason         string
	ReviewCharacterLabel string
	ReviewBadgeClass     string
	Viewer               bool
}

// CampaignChatOOCPostView carries one append-only OOC post.
type CampaignChatOOCPostView struct {
	ParticipantID string
	Body          string
	Viewer        bool
}

// campaignChatName prefers the rendered campaign label and falls back to the route id.
func campaignChatName(view CampaignChatView) string {
	if strings.TrimSpace(view.CampaignName) != "" {
		return strings.TrimSpace(view.CampaignName)
	}
	if strings.TrimSpace(view.CampaignID) != "" {
		return strings.TrimSpace(view.CampaignID)
	}
	return "Campaign"
}

// campaignChatTitle derives the browser title from the current campaign label.
func campaignChatTitle(view CampaignChatView) string {
	return campaignChatName(view) + " Game"
}

// campaignChatBackURL keeps the chat surface on canonical campaign routes.
func campaignChatBackURL(view CampaignChatView) string {
	if strings.TrimSpace(view.BackURL) != "" {
		return strings.TrimSpace(view.BackURL)
	}
	if strings.TrimSpace(view.CampaignID) != "" {
		return routepath.AppCampaign(strings.TrimSpace(view.CampaignID))
	}
	return routepath.AppCampaigns
}

// campaignChatParticipantLabel keeps participant copy stable with safe fallback text.
func campaignChatParticipantLabel(view CampaignChatView) string {
	if strings.TrimSpace(view.ParticipantName) != "" {
		return strings.TrimSpace(view.ParticipantName)
	}
	return "Participant"
}

// campaignChatSessionLabel keeps session copy stable with safe fallback text.
func campaignChatSessionLabel(view CampaignChatView) string {
	if strings.TrimSpace(view.SessionName) != "" {
		return strings.TrimSpace(view.SessionName)
	}
	return "No active session"
}

// campaignChatSceneLabel keeps scene copy stable with safe fallback text.
func campaignChatSceneLabel(view CampaignChatView) string {
	if strings.TrimSpace(view.SceneName) != "" {
		return strings.TrimSpace(view.SceneName)
	}
	return "No active scene"
}

// campaignChatPhaseSummary keeps the current scene-phase summary readable when empty.
func campaignChatPhaseSummary(view CampaignChatView) string {
	if strings.TrimSpace(view.PhaseLabel) != "" {
		return strings.TrimSpace(view.PhaseLabel)
	}
	return "GM turn"
}

// campaignChatOOCSummary keeps the current OOC summary readable when empty.
func campaignChatOOCSummary(view CampaignChatView) string {
	if strings.TrimSpace(view.OOCSummary) != "" {
		return strings.TrimSpace(view.OOCSummary)
	}
	return "In character"
}
