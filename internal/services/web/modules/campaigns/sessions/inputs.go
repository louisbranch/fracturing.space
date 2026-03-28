package sessions

import (
	"net/url"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// parseStartSessionInput normalizes the session-create form into app input.
func parseStartSessionInput(form url.Values) campaignapp.StartSessionInput {
	characterIDs := form["character_controller_character_id"]
	participantIDs := form["character_controller_participant_id"]
	assignments := make([]campaignapp.SessionCharacterControllerAssignment, 0, len(characterIDs))
	for idx, characterID := range characterIDs {
		participantID := ""
		if idx < len(participantIDs) {
			participantID = participantIDs[idx]
		}
		assignments = append(assignments, campaignapp.SessionCharacterControllerAssignment{
			CharacterID:   strings.TrimSpace(characterID),
			ParticipantID: strings.TrimSpace(participantID),
		})
	}
	return campaignapp.StartSessionInput{
		Name:                 strings.TrimSpace(form.Get("name")),
		CharacterControllers: assignments,
	}
}

// parseEndSessionInput normalizes the session-end form into app input.
func parseEndSessionInput(form url.Values) campaignapp.EndSessionInput {
	return campaignapp.EndSessionInput{SessionID: strings.TrimSpace(form.Get("session_id"))}
}
