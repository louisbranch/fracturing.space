package campaigns

import (
	"encoding/json"
	"net/http"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- Campaign chat route ---

// handleGame renders the chat view for a campaign workspace route.
func (h handlers) handleGame(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, err := h.loadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	surface, err := h.service.CampaignGameSurface(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	bootstrapJSON, err := json.Marshal(surface)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := CampaignChatView{
		CampaignID:       campaignID,
		CampaignName:     page.workspace.Name,
		BackURL:          routepath.AppCampaign(campaignID),
		ChatFallbackPort: strings.TrimSpace(h.chatFallbackPort),
		BootstrapJSON:    string(bootstrapJSON),
		ParticipantName:  surface.Participant.Name,
		ParticipantRole:  surface.Participant.Role,
		SessionName:      surface.SessionName,
		DefaultStreamID:  surface.DefaultStreamID,
		DefaultPersonaID: surface.DefaultPersonaID,
		GateSummary:      campaignGameGateSummary(surface.ActiveSessionGate),
		SpotlightSummary: campaignGameSpotlightSummary(surface.ActiveSessionSpotlight),
		ActiveGateType:   activeGateType(surface.ActiveSessionGate),
		ActiveGateStatus: activeGateStatus(surface.ActiveSessionGate),
		Streams:          campaignGameStreamViews(surface.Streams, surface.DefaultStreamID),
		Personas:         campaignGamePersonaViews(surface.Personas, surface.DefaultPersonaID),
	}
	h.writeCampaignChatHTML(w, r, view, page.lang, page.loc)
}

// writeCampaignChatHTML centralizes this web behavior in one helper seam.
func (h handlers) writeCampaignChatHTML(
	w http.ResponseWriter,
	r *http.Request,
	view CampaignChatView,
	lang string,
	loc Localizer,
) {
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, routepath.AppCampaignGame(view.CampaignID))
		return
	}
	if err := CampaignChatPage(view, lang, loc).Render(r.Context(), w); err != nil {
		h.WriteError(w, r, err)
	}
}

// campaignGameStreamViews maps app-layer stream context into the template view
// model so the page renders authoritative game-owned routing metadata.
func campaignGameStreamViews(streams []campaignapp.CampaignGameStream, defaultStreamID string) []CampaignChatStreamView {
	if len(streams) == 0 {
		return []CampaignChatStreamView{}
	}
	views := make([]CampaignChatStreamView, 0, len(streams))
	defaultStreamID = strings.TrimSpace(defaultStreamID)
	for _, stream := range streams {
		label := strings.TrimSpace(stream.Label)
		if label == "" {
			label = strings.TrimSpace(stream.ID)
		}
		secondary := strings.TrimSpace(stream.Scope)
		if strings.TrimSpace(stream.SceneID) != "" {
			secondary = "scene · " + strings.TrimSpace(stream.SceneID)
		} else if strings.TrimSpace(stream.SessionID) != "" {
			secondary = "session · " + strings.TrimSpace(stream.SessionID)
		}
		views = append(views, CampaignChatStreamView{
			StreamID:      strings.TrimSpace(stream.ID),
			Label:         label,
			Kind:          strings.TrimSpace(stream.Kind),
			Scope:         strings.TrimSpace(stream.Scope),
			SecondaryText: secondary,
			Active:        strings.TrimSpace(stream.ID) == defaultStreamID,
		})
	}
	return views
}

// campaignGamePersonaViews maps app-layer persona options into the template
// view model so the surface can render allowed speaking identities.
func campaignGamePersonaViews(personas []campaignapp.CampaignGamePersona, defaultPersonaID string) []CampaignChatPersonaView {
	if len(personas) == 0 {
		return []CampaignChatPersonaView{}
	}
	views := make([]CampaignChatPersonaView, 0, len(personas))
	defaultPersonaID = strings.TrimSpace(defaultPersonaID)
	for _, persona := range personas {
		displayName := strings.TrimSpace(persona.DisplayName)
		if displayName == "" {
			displayName = strings.TrimSpace(persona.ID)
		}
		kindLabel := strings.TrimSpace(persona.Kind)
		if strings.TrimSpace(persona.CharacterID) != "" {
			kindLabel = strings.TrimSpace(persona.Kind) + " · " + strings.TrimSpace(persona.CharacterID)
		}
		views = append(views, CampaignChatPersonaView{
			PersonaID:   strings.TrimSpace(persona.ID),
			DisplayName: displayName,
			KindLabel:   kindLabel,
			Active:      strings.TrimSpace(persona.ID) == defaultPersonaID,
		})
	}
	return views
}

// campaignGameGateSummary provides a compact reader-facing gate label for the
// initial server-rendered game surface.
func campaignGameGateSummary(gate *campaignapp.CampaignGameGate) string {
	if gate == nil {
		return "No active gate"
	}
	summary := strings.TrimSpace(gate.Type)
	if summary == "" {
		summary = "gate"
	}
	if strings.TrimSpace(gate.Status) != "" {
		summary += " · " + strings.TrimSpace(gate.Status)
	}
	if strings.TrimSpace(gate.Reason) != "" {
		summary += " · " + strings.TrimSpace(gate.Reason)
	}
	return summary
}

// campaignGameSpotlightSummary provides a compact reader-facing spotlight label
// for the initial server-rendered game surface.
func campaignGameSpotlightSummary(spotlight *campaignapp.CampaignGameSpotlight) string {
	if spotlight == nil {
		return "No active spotlight"
	}
	summary := strings.TrimSpace(spotlight.Type)
	if summary == "" {
		summary = "spotlight"
	}
	if strings.TrimSpace(spotlight.CharacterID) != "" {
		summary += " · " + strings.TrimSpace(spotlight.CharacterID)
	}
	return summary
}

// activeGateType exposes the normalized active gate type for template state.
func activeGateType(gate *campaignapp.CampaignGameGate) string {
	if gate == nil {
		return ""
	}
	return strings.TrimSpace(gate.Type)
}

// activeGateStatus exposes the normalized active gate status for template
// state.
func activeGateStatus(gate *campaignapp.CampaignGameGate) string {
	if gate == nil {
		return ""
	}
	return strings.TrimSpace(gate.Status)
}
