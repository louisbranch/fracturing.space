package web

import (
	"html"
	"io"
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

func (h *handler) handleAppCampaignParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignParticipants renders participant read models only after
	// explicit campaign membership is verified.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireCampaignParticipant(w, r, campaignID) {
		return
	}
	if h.participantClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Participants unavailable", "participant service client is not configured")
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	canManageParticipants := false
	if participant, err := h.campaignParticipant(r.Context(), campaignID, sess.accessToken); err == nil && participant != nil && strings.TrimSpace(participant.GetId()) != "" {
		canManageParticipants = canManageCampaignParticipants(participant.GetCampaignAccess())
	}

	resp, err := h.participantClient.ListParticipants(r.Context(), &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Participants unavailable", "failed to list participants")
		return
	}

	renderAppCampaignParticipantsPage(w, campaignID, resp.GetParticipants(), canManageParticipants)
}

func (h *handler) handleAppCampaignParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignParticipantUpdate applies participant role/access/controller changes
	// for managers and owners only, preserving campaign governance semantics.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireCampaignParticipant(w, r, campaignID) {
		return
	}
	if h.participantClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Participant action unavailable", "participant service client is not configured")
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "failed to parse participant update form")
		return
	}
	targetParticipantID := strings.TrimSpace(r.FormValue("participant_id"))
	if targetParticipantID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "participant id is required")
		return
	}
	updateReq := &statev1.UpdateParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: targetParticipantID,
	}
	hasFieldUpdate := false
	if rawAccess := strings.TrimSpace(r.FormValue("campaign_access")); rawAccess != "" {
		targetAccess, ok := parseCampaignAccessFormValue(rawAccess)
		if !ok {
			h.renderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "campaign access value is invalid")
			return
		}
		updateReq.CampaignAccess = targetAccess
		hasFieldUpdate = true
	}
	if rawRole := strings.TrimSpace(r.FormValue("role")); rawRole != "" {
		targetRole, ok := parseParticipantRoleFormValue(rawRole)
		if !ok {
			h.renderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "participant role value is invalid")
			return
		}
		updateReq.Role = targetRole
		hasFieldUpdate = true
	}
	if rawController := strings.TrimSpace(r.FormValue("controller")); rawController != "" {
		targetController, ok := parseParticipantControllerFormValue(rawController)
		if !ok {
			h.renderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "participant controller value is invalid")
			return
		}
		updateReq.Controller = targetController
		hasFieldUpdate = true
	}
	if !hasFieldUpdate {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "at least one participant field is required")
		return
	}

	actingParticipant, err := h.campaignParticipant(r.Context(), campaignID, sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Participant action unavailable", "failed to resolve campaign participant")
		return
	}
	actingParticipantID := ""
	if actingParticipant != nil {
		actingParticipantID = strings.TrimSpace(actingParticipant.GetId())
	}
	if actingParticipantID == "" {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant identity required for participant action")
		return
	}
	if !canManageCampaignParticipants(actingParticipant.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for participant action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), actingParticipantID)
	_, err = h.participantClient.UpdateParticipant(ctx, updateReq)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Participant action unavailable", "failed to update participant")
		return
	}

	http.Redirect(w, r, "/app/campaigns/"+campaignID+"/participants", http.StatusFound)
}

func canManageCampaignParticipants(access statev1.CampaignAccess) bool {
	return access == statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER || access == statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
}

func parseCampaignAccessFormValue(raw string) (statev1.CampaignAccess, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "member", "campaign_access_member":
		return statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER, true
	case "manager", "campaign_access_manager":
		return statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER, true
	case "owner", "campaign_access_owner":
		return statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER, true
	default:
		return statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED, false
	}
}

func campaignAccessFormValue(access statev1.CampaignAccess) string {
	switch access {
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return "manager"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return "owner"
	default:
		return "member"
	}
}

func parseParticipantRoleFormValue(raw string) (statev1.ParticipantRole, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "gm", "participant_role_gm":
		return statev1.ParticipantRole_GM, true
	case "player", "participant_role_player":
		return statev1.ParticipantRole_PLAYER, true
	default:
		return statev1.ParticipantRole_ROLE_UNSPECIFIED, false
	}
}

func participantRoleFormValue(role statev1.ParticipantRole) string {
	if role == statev1.ParticipantRole_GM {
		return "gm"
	}
	return "player"
}

func parseParticipantControllerFormValue(raw string) (statev1.Controller, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "human", "controller_human":
		return statev1.Controller_CONTROLLER_HUMAN, true
	case "ai", "controller_ai":
		return statev1.Controller_CONTROLLER_AI, true
	default:
		return statev1.Controller_CONTROLLER_UNSPECIFIED, false
	}
}

func participantControllerFormValue(controller statev1.Controller) string {
	if controller == statev1.Controller_CONTROLLER_AI {
		return "ai"
	}
	return "human"
}

func renderAppCampaignParticipantsPage(w http.ResponseWriter, campaignID string, participants []*statev1.Participant, canManageParticipants bool) {
	// renderAppCampaignParticipantsPage translates participant domain objects into the
	// management controls available at this campaign membership level.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	escapedCampaignID := html.EscapeString(campaignID)
	_, _ = io.WriteString(w, "<!doctype html><html><head><title>Participants</title></head><body><h1>Participants</h1><ul>")
	for _, participant := range participants {
		if participant == nil {
			continue
		}
		name := strings.TrimSpace(participant.GetName())
		if name == "" {
			name = strings.TrimSpace(participant.GetUserId())
		}
		if name == "" {
			name = strings.TrimSpace(participant.GetId())
		}
		_, _ = io.WriteString(w, "<li>"+html.EscapeString(name))
		if canManageParticipants {
			participantID := strings.TrimSpace(participant.GetId())
			if participantID != "" {
				selectedAccess := campaignAccessFormValue(participant.GetCampaignAccess())
				selectedRole := participantRoleFormValue(participant.GetRole())
				selectedController := participantControllerFormValue(participant.GetController())
				memberSelected := ""
				managerSelected := ""
				ownerSelected := ""
				switch selectedAccess {
				case "manager":
					managerSelected = " selected"
				case "owner":
					ownerSelected = " selected"
				default:
					memberSelected = " selected"
				}
				gmSelected := ""
				playerSelected := ""
				if selectedRole == "gm" {
					gmSelected = " selected"
				} else {
					playerSelected = " selected"
				}
				humanSelected := ""
				aiSelected := ""
				if selectedController == "ai" {
					aiSelected = " selected"
				} else {
					humanSelected = " selected"
				}
				_, _ = io.WriteString(w, "<form method=\"post\" action=\"/app/campaigns/"+escapedCampaignID+"/participants/update\"><input type=\"hidden\" name=\"participant_id\" value=\""+html.EscapeString(participantID)+"\"><select name=\"campaign_access\"><option value=\"member\""+memberSelected+">member</option><option value=\"manager\""+managerSelected+">manager</option><option value=\"owner\""+ownerSelected+">owner</option></select><select name=\"role\"><option value=\"gm\""+gmSelected+">gm</option><option value=\"player\""+playerSelected+">player</option></select><select name=\"controller\"><option value=\"human\""+humanSelected+">human</option><option value=\"ai\""+aiSelected+">ai</option></select><button type=\"submit\">Update Access</button></form>")
			}
		}
		_, _ = io.WriteString(w, "</li>")
	}
	_, _ = io.WriteString(w, "</ul></body></html>")
}
