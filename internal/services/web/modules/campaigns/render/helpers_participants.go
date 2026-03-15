package render

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// campaignParticipantEditURL keeps participant edit links consistent with routepath.
func campaignParticipantEditURL(campaignID string, participant ParticipantView) string {
	campaignID = strings.TrimSpace(campaignID)
	participantID := strings.TrimSpace(participant.ID)
	if campaignID == "" || participantID == "" {
		return ""
	}
	return routepath.AppCampaignParticipantEdit(campaignID, participantID)
}

// campaignParticipantPronounPresets keeps participant-edit suggestions in the render seam.
func campaignParticipantPronounPresets(loc Localizer, editor ParticipantEditorView) []string {
	presets := []string{
		T(loc, "game.participants.value_she_her"),
		T(loc, "game.participants.value_he_him"),
		T(loc, "game.participants.value_they_them"),
	}
	if campaignParticipantControllerCanonical(editor.Controller) == "ai" {
		return append(presets, T(loc, "game.participants.value_it_its"))
	}
	return presets
}

// campaignParticipantControllerCanonical normalizes controller values from backend and form inputs.
func campaignParticipantControllerCanonical(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ai", "controller_ai":
		return "ai"
	case "human", "controller_human":
		return "human"
	case "unassigned", "controller_unassigned":
		return "unassigned"
	default:
		return ""
	}
}

// campaignParticipantRoleFormValue ensures participant edit forms always submit a concrete role.
func campaignParticipantRoleFormValue(editor ParticipantEditorView) string {
	if value := campaignParticipantRoleCanonical(editor.Role); value != "" {
		return value
	}
	return "gm"
}

// campaignParticipantRoleCanonical normalizes participant role values for comparisons and forms.
func campaignParticipantRoleCanonical(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gm", "participant_role_gm", "role_gm":
		return "gm"
	case "player", "participant_role_player", "role_player":
		return "player"
	default:
		return ""
	}
}

// campaignParticipantAccessCanonical normalizes participant access values for comparisons and forms.
func campaignParticipantAccessCanonical(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "member", "campaign_access_member":
		return "member"
	case "manager", "campaign_access_manager":
		return "manager"
	case "owner", "campaign_access_owner":
		return "owner"
	default:
		return ""
	}
}

// campaignParticipantAccessFormValue ensures participant edit forms always submit a concrete access value.
func campaignParticipantAccessFormValue(editor ParticipantEditorView) string {
	if value := campaignParticipantAccessCanonical(editor.CampaignAccess); value != "" {
		return value
	}
	return "member"
}
