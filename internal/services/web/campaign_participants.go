package web

import (
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) handleAppCampaignParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignParticipants renders participant read models only after
	// explicit campaign membership is verified.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	participant, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if h.participantClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Participants unavailable", "participant service client is not configured")
		return
	}
	canManageParticipants := canManageCampaignParticipants(participant.GetCampaignAccess())

	resp, err := h.participantClient.ListParticipants(r.Context(), &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Participants unavailable", "failed to list participants")
		return
	}

	renderAppCampaignParticipantsPageWithContext(w, r, h.pageContextForCampaign(w, r, campaignID), campaignID, resp.GetParticipants(), canManageParticipants)
}

func (h *handler) handleAppCampaignParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignParticipantUpdate applies participant role/access/controller changes
	// for managers and owners only, preserving campaign governance semantics.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actingParticipant, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if h.participantClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Participant action unavailable", "participant service client is not configured")
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

	if !canManageCampaignParticipants(actingParticipant.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for participant action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actingParticipant.GetId()))
	_, err := h.participantClient.UpdateParticipant(ctx, updateReq)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Participant action unavailable", "failed to update participant")
		return
	}

	http.Redirect(w, r, "/campaigns/"+campaignID+"/participants", http.StatusFound)
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

func renderAppCampaignParticipantsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, participants []*statev1.Participant, canManageParticipants bool) {
	renderAppCampaignParticipantsPageWithContext(w, r, page, campaignID, participants, canManageParticipants)
}

func renderAppCampaignParticipantsPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, participants []*statev1.Participant, canManageParticipants bool) {
	// renderAppCampaignParticipantsPage translates participant domain objects into the
	// management controls available at this campaign membership level.
	campaignID = strings.TrimSpace(campaignID)
	participantItems := make([]webtemplates.ParticipantListItem, 0, len(participants))
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
		accessValue := campaignAccessFormValue(participant.GetCampaignAccess())
		roleValue := participantRoleFormValue(participant.GetRole())
		controllerValue := participantControllerFormValue(participant.GetController())
		participantItems = append(participantItems, webtemplates.ParticipantListItem{
			ID:              strings.TrimSpace(participant.GetId()),
			Name:            name,
			MemberSelected:  accessValue == "member",
			ManagerSelected: accessValue == "manager",
			OwnerSelected:   accessValue == "owner",
			GMSelected:      roleValue == "gm",
			PlayerSelected:  roleValue == "player",
			HumanSelected:   controllerValue == "human",
			AISelected:      controllerValue == "ai",
		})
	}
	writeGameContentType(w)
	if err := webtemplates.CampaignParticipantsPage(page, campaignID, canManageParticipants, participantItems).Render(r.Context(), w); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_participants_page")
	}
}
