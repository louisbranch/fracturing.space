package render

import (
	"strings"

	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// campaignSessionStatusLabel keeps session tables and detail pages on the same status copy.
func campaignSessionStatusLabel(loc webtemplates.Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return webtemplates.T(loc, "game.campaign.session_status_unspecified")
	case "active":
		return webtemplates.T(loc, "game.campaign.session_status_active")
	case "ended":
		return webtemplates.T(loc, "game.campaign.session_status_ended")
	default:
		return raw
	}
}

// campaignSessionCanEnd gates end-session affordances to active sessions only.
func campaignSessionCanEnd(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "active")
}

// campaignSessionStartReady exposes the session-readiness contract to templates.
func campaignSessionStartReady(readiness SessionReadinessView) bool {
	return readiness.Ready
}

// sessionCreateHasCompleteControllerAssignments reports whether every
// character row currently has a selected controller.
func sessionCreateHasCompleteControllerAssignments(view SessionCreatePageView) bool {
	for _, character := range view.CharacterControllers {
		hasSelection := false
		for _, option := range character.Options {
			if option.Selected && strings.TrimSpace(option.ParticipantID) != "" {
				hasSelection = true
				break
			}
		}
		if !hasSelection {
			return false
		}
	}
	return true
}

// campaignSessionControllerOptionLabel keeps session-start controller labels stable.
func campaignSessionControllerOptionLabel(loc webtemplates.Localizer, option SessionCreateControllerOptionView) string {
	if strings.TrimSpace(option.ParticipantID) == "" {
		return webtemplates.T(loc, "game.participants.value_unassigned")
	}
	label := strings.TrimSpace(option.Label)
	if label == "" {
		return strings.TrimSpace(option.ParticipantID)
	}
	return label
}

// campaignSessionByID resolves the selected session without forcing handlers to pre-split the view.
func campaignSessionByID(_ webtemplates.Localizer, sessionID string, sessions []SessionView) SessionView {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionView{}
	}
	for _, session := range sessions {
		if strings.TrimSpace(session.ID) == sessionID {
			return session
		}
	}
	return SessionView{}
}
