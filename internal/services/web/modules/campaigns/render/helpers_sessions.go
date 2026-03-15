package render

import "strings"

// campaignSessionStatusLabel keeps session tables and detail pages on the same status copy.
func campaignSessionStatusLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return T(loc, "game.campaign.session_status_unspecified")
	case "active":
		return T(loc, "game.campaign.session_status_active")
	case "ended":
		return T(loc, "game.campaign.session_status_ended")
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

// campaignSessionByID resolves the selected session without forcing handlers to pre-split the view.
func campaignSessionByID(_ Localizer, sessionID string, sessions []SessionView) SessionView {
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
