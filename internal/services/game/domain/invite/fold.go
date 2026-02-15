package invite

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Fold applies an event to invite state.
func Fold(state State, evt event.Event) State {
	if evt.Type == eventTypeCreated {
		state.Created = true
		var payload CreatePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		state.InviteID = payload.InviteID
		state.ParticipantID = payload.ParticipantID
		state.RecipientUserID = payload.RecipientUserID
		state.CreatedByParticipantID = payload.CreatedByParticipantID
		status := payload.Status
		if normalized, ok := normalizeStatusLabel(payload.Status); ok {
			status = normalized
		}
		state.Status = status
	}
	if evt.Type == eventTypeClaimed {
		var payload ClaimPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.InviteID != "" {
			state.InviteID = payload.InviteID
		}
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
		state.Status = statusClaimed
	}
	if evt.Type == eventTypeRevoked {
		var payload RevokePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.InviteID != "" {
			state.InviteID = payload.InviteID
		}
		state.Status = statusRevoked
	}
	if evt.Type == eventTypeUpdated {
		var payload UpdatePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.InviteID != "" {
			state.InviteID = payload.InviteID
		}
		if payload.Status != "" {
			status := payload.Status
			if normalized, ok := normalizeStatusLabel(payload.Status); ok {
				status = normalized
			}
			state.Status = status
		}
	}
	return state
}
