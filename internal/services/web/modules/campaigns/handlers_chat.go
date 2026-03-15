package campaigns

import (
	"encoding/json"
	"net/http"
	"strconv"
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
	surface, err := h.game.CampaignGameSurface(ctx, campaignID)
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
		CampaignID:           campaignID,
		CampaignName:         page.workspace.Name,
		BackURL:              routepath.AppCampaign(campaignID),
		BootstrapJSON:        string(bootstrapJSON),
		ParticipantName:      surface.Participant.Name,
		ParticipantRole:      surface.Participant.Role,
		SessionName:          surface.SessionName,
		SceneName:            campaignGameSceneName(surface.ActiveScene),
		SceneDescription:     campaignGameSceneDescription(surface.ActiveScene),
		PhaseStatus:          campaignGamePhaseStatus(surface.PlayerPhase),
		PhaseLabel:           campaignGamePhaseLabel(surface.PlayerPhase),
		PhaseFrame:           campaignGamePhaseFrame(surface.PlayerPhase),
		OOCSummary:           campaignGameOOCSummary(surface.OOC),
		HasOpenOOC:           surface.OOC.Open,
		ActingCharacters:     campaignGameCharacterViews(surface.ActiveScene, surface.PlayerPhase),
		SceneCharacters:      campaignGameSceneCharacters(surface.ActiveScene),
		Slots:                campaignGameSlotViews(surface),
		OOCPosts:             campaignGameOOCPostViews(surface),
		YieldedParticipants:  append([]string(nil), campaignGameYieldedParticipants(surface)...),
		OOCReadyParticipants: append([]string(nil), surface.OOC.ReadyToResumeParticipantIDs...),
		GMAuthorityLabel:     campaignGameGMAuthorityLabel(surface),
		AITurnStatus:         campaignGameAITurnStatus(surface),
		AITurnSummary:        campaignGameAITurnSummary(surface),
		AITurnError:          campaignGameAITurnError(surface),
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

// campaignGameSceneName keeps template fallbacks centralized for scene titles.
func campaignGameSceneName(scene *campaignapp.CampaignGameScene) string {
	if scene == nil {
		return ""
	}
	return strings.TrimSpace(scene.Name)
}

// campaignGameSceneDescription trims optional scene copy for presentation.
func campaignGameSceneDescription(scene *campaignapp.CampaignGameScene) string {
	if scene == nil {
		return ""
	}
	return strings.TrimSpace(scene.Description)
}

// campaignGamePhaseLabel maps interaction phase state into page copy.
func campaignGamePhaseLabel(phase *campaignapp.CampaignGamePlayerPhase) string {
	if phase == nil {
		return "GM turn"
	}
	switch campaignGamePhaseStatus(phase) {
	case "players":
		return "Players acting"
	case "gm_review":
		return "GM reviewing"
	case "gm":
		return "GM turn"
	default:
		return "Scene phase"
	}
}

// campaignGamePhaseStatus keeps phase-status fallback logic in one view seam.
func campaignGamePhaseStatus(phase *campaignapp.CampaignGamePlayerPhase) string {
	if phase == nil {
		return "gm"
	}
	status := strings.TrimSpace(phase.Status)
	if status == "" {
		return "gm"
	}
	return status
}

// campaignGamePhaseFrame exposes the current scene frame text for the page.
func campaignGamePhaseFrame(phase *campaignapp.CampaignGamePlayerPhase) string {
	if phase == nil {
		return ""
	}
	return strings.TrimSpace(phase.FrameText)
}

// campaignGameOOCSummary condenses OOC state into a small table-status label.
func campaignGameOOCSummary(ooc campaignapp.CampaignGameOOCState) string {
	if !ooc.Open {
		return "In character"
	}
	if ready := len(ooc.ReadyToResumeParticipantIDs); ready > 0 {
		return "OOC paused · ready " + strconv.Itoa(ready)
	}
	return "OOC paused"
}

// campaignGameGMAuthorityLabel keeps GM-owner copy stable for the game page.
func campaignGameGMAuthorityLabel(surface campaignapp.CampaignGameSurface) string {
	if participantID := strings.TrimSpace(surface.GMAuthorityParticipantID); participantID != "" {
		return participantID
	}
	return "Unassigned"
}

// campaignGameAITurnStatus centralizes the page-level AI-turn status fallback.
func campaignGameAITurnStatus(surface campaignapp.CampaignGameSurface) string {
	status := strings.TrimSpace(surface.AITurn.Status)
	if status == "" {
		return "idle"
	}
	return status
}

// campaignGameAITurnSummary condenses AI-turn state into a small status label.
func campaignGameAITurnSummary(surface campaignapp.CampaignGameSurface) string {
	switch campaignGameAITurnStatus(surface) {
	case "queued":
		return "Queued for AI GM resolution"
	case "running":
		return "AI GM is resolving the scene"
	case "failed":
		return "AI GM turn failed"
	case "idle":
		return "No AI GM turn queued"
	default:
		return "AI GM state unavailable"
	}
}

// campaignGameAITurnError exposes the latest AI-turn failure copy for the page.
func campaignGameAITurnError(surface campaignapp.CampaignGameSurface) string {
	return strings.TrimSpace(surface.AITurn.LastError)
}

// campaignGameSceneCharacters maps scene roster data into template-ready views.
func campaignGameSceneCharacters(scene *campaignapp.CampaignGameScene) []CampaignChatCharacterView {
	if scene == nil || len(scene.Characters) == 0 {
		return []CampaignChatCharacterView{}
	}
	views := make([]CampaignChatCharacterView, 0, len(scene.Characters))
	for _, character := range scene.Characters {
		views = append(views, CampaignChatCharacterView{
			CharacterID:        strings.TrimSpace(character.ID),
			Name:               firstNonEmpty(character.Name, character.ID),
			OwnerParticipantID: strings.TrimSpace(character.OwnerParticipantID),
		})
	}
	return views
}

// campaignGameCharacterViews annotates the scene roster with active acting state.
func campaignGameCharacterViews(scene *campaignapp.CampaignGameScene, phase *campaignapp.CampaignGamePlayerPhase) []CampaignChatCharacterView {
	views := campaignGameSceneCharacters(scene)
	if len(views) == 0 || phase == nil {
		return views
	}
	active := make(map[string]struct{}, len(phase.ActingCharacterIDs))
	for _, characterID := range phase.ActingCharacterIDs {
		characterID = strings.TrimSpace(characterID)
		if characterID == "" {
			continue
		}
		active[characterID] = struct{}{}
	}
	for i := range views {
		_, views[i].Active = active[views[i].CharacterID]
	}
	return views
}

// campaignGameSlotViews projects participant-owned slots into page cards.
func campaignGameSlotViews(surface campaignapp.CampaignGameSurface) []CampaignChatPlayerSlotView {
	if surface.PlayerPhase == nil || len(surface.PlayerPhase.Slots) == 0 {
		return []CampaignChatPlayerSlotView{}
	}
	characterNames := make(map[string]string)
	if surface.ActiveScene != nil {
		for _, character := range surface.ActiveScene.Characters {
			characterNames[strings.TrimSpace(character.ID)] = firstNonEmpty(character.Name, character.ID)
		}
	}
	views := make([]CampaignChatPlayerSlotView, 0, len(surface.PlayerPhase.Slots))
	for _, slot := range surface.PlayerPhase.Slots {
		characters := make([]string, 0, len(slot.CharacterIDs))
		for _, characterID := range slot.CharacterIDs {
			characterID = strings.TrimSpace(characterID)
			if characterID == "" {
				continue
			}
			characters = append(characters, firstNonEmpty(characterNames[characterID], characterID))
		}
		reviewCharacters := make([]string, 0, len(slot.ReviewCharacterIDs))
		for _, characterID := range slot.ReviewCharacterIDs {
			characterID = strings.TrimSpace(characterID)
			if characterID == "" {
				continue
			}
			reviewCharacters = append(reviewCharacters, firstNonEmpty(characterNames[characterID], characterID))
		}
		participantID := strings.TrimSpace(slot.ParticipantID)
		reviewStatus := normalizeCampaignGameSlotReviewStatus(slot.ReviewStatus)
		reviewLabel, reviewBadgeClass := campaignGameSlotReviewLabel(reviewStatus)
		views = append(views, CampaignChatPlayerSlotView{
			ParticipantID:        participantID,
			SummaryText:          strings.TrimSpace(slot.SummaryText),
			CharacterLabel:       strings.Join(characters, ", "),
			Yielded:              slot.Yielded,
			ReviewStatus:         reviewStatus,
			ReviewLabel:          reviewLabel,
			ReviewReason:         strings.TrimSpace(slot.ReviewReason),
			ReviewCharacterLabel: strings.Join(reviewCharacters, ", "),
			ReviewBadgeClass:     reviewBadgeClass,
			Viewer:               participantID == strings.TrimSpace(surface.Participant.ID),
		})
	}
	return views
}

// campaignGameOOCPostViews maps the session OOC overlay into transcript rows.
func campaignGameOOCPostViews(surface campaignapp.CampaignGameSurface) []CampaignChatOOCPostView {
	if len(surface.OOC.Posts) == 0 {
		return []CampaignChatOOCPostView{}
	}
	views := make([]CampaignChatOOCPostView, 0, len(surface.OOC.Posts))
	for _, post := range surface.OOC.Posts {
		participantID := strings.TrimSpace(post.ParticipantID)
		views = append(views, CampaignChatOOCPostView{
			ParticipantID: participantID,
			Body:          strings.TrimSpace(post.Body),
			Viewer:        participantID == strings.TrimSpace(surface.Participant.ID),
		})
	}
	return views
}

// firstNonEmpty prefers explicit labels while preserving stable ID fallbacks.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// normalizeCampaignGameSlotReviewStatus keeps slot-review fallbacks consistent.
func normalizeCampaignGameSlotReviewStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "open"
	}
	return status
}

// campaignGameSlotReviewLabel maps review state into compact badge copy.
func campaignGameSlotReviewLabel(status string) (label string, badgeClass string) {
	switch normalizeCampaignGameSlotReviewStatus(status) {
	case "under_review":
		return "Under review", "badge-outline"
	case "accepted":
		return "Accepted", "badge-success"
	case "changes_requested":
		return "Changes requested", "badge-warning"
	default:
		return "Open", "badge-ghost"
	}
}

// campaignGameYieldedParticipants exposes yielded participants for the page state.
func campaignGameYieldedParticipants(surface campaignapp.CampaignGameSurface) []string {
	if surface.PlayerPhase == nil {
		return []string{}
	}
	yielded := make([]string, 0, len(surface.PlayerPhase.Slots))
	for _, slot := range surface.PlayerPhase.Slots {
		if strings.TrimSpace(slot.ParticipantID) == "" || !slot.Yielded {
			continue
		}
		yielded = append(yielded, strings.TrimSpace(slot.ParticipantID))
	}
	return yielded
}
