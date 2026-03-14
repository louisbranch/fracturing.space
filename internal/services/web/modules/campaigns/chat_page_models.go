package campaigns

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// CampaignChatView carries the game chat page state for one campaign workspace.
type CampaignChatView struct {
	CampaignID       string
	CampaignName     string
	BackURL          string
	ChatFallbackPort string
	BootstrapJSON    string
	ParticipantName  string
	ParticipantRole  string
	SessionName      string
	DefaultStreamID  string
	DefaultPersonaID string
	GateSummary      string
	SpotlightSummary string
	ActiveGateType   string
	ActiveGateStatus string
	Streams          []CampaignChatStreamView
	Personas         []CampaignChatPersonaView
}

// CampaignChatStreamView carries one stream selector row for the chat surface.
type CampaignChatStreamView struct {
	StreamID      string
	Label         string
	Kind          string
	Scope         string
	SecondaryText string
	Active        bool
}

// CampaignChatPersonaView carries one persona selector option for the chat surface.
type CampaignChatPersonaView struct {
	PersonaID   string
	DisplayName string
	KindLabel   string
	Active      bool
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

// campaignChatFallbackPort exposes the websocket fallback port to the page shell.
func campaignChatFallbackPort(view CampaignChatView) string {
	return strings.TrimSpace(view.ChatFallbackPort)
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

// campaignChatGateSummary keeps the current gate summary readable when empty.
func campaignChatGateSummary(view CampaignChatView) string {
	if strings.TrimSpace(view.GateSummary) != "" {
		return strings.TrimSpace(view.GateSummary)
	}
	return "No active gate"
}

// campaignChatSpotlightSummary keeps the current spotlight summary readable when empty.
func campaignChatSpotlightSummary(view CampaignChatView) string {
	if strings.TrimSpace(view.SpotlightSummary) != "" {
		return strings.TrimSpace(view.SpotlightSummary)
	}
	return "No active spotlight"
}

// campaignChatStreamButtonClass preserves the active/inactive stream control styling.
func campaignChatStreamButtonClass(stream CampaignChatStreamView) string {
	if stream.Active {
		return "btn btn-sm btn-primary w-full justify-between"
	}
	return "btn btn-sm btn-ghost w-full justify-between"
}
