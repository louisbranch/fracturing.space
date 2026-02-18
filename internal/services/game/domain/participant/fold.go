package participant

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Fold applies an event to participant state.
func Fold(state State, evt event.Event) State {
	if evt.Type == eventTypeJoined {
		state.Joined = true
		state.Left = false
		var payload JoinPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		state.ParticipantID = payload.ParticipantID
		state.UserID = payload.UserID
		state.Name = payload.Name
		state.Role = payload.Role
		state.Controller = payload.Controller
		state.CampaignAccess = payload.CampaignAccess
	}
	if evt.Type == eventTypeUpdated {
		var payload UpdatePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
		for key, value := range payload.Fields {
			switch key {
			case "user_id":
				state.UserID = value
			case "name":
				state.Name = value
			case "role":
				state.Role = value
			case "controller":
				state.Controller = value
			case "campaign_access":
				state.CampaignAccess = value
			}
		}
	}
	if evt.Type == eventTypeLeft {
		state.Left = true
		state.Joined = false
		var payload LeavePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
	}
	if evt.Type == eventTypeBound {
		var payload BindPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
		state.UserID = payload.UserID
	}
	if evt.Type == eventTypeUnbound {
		var payload UnbindPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
		state.UserID = ""
	}
	if evt.Type == eventTypeSeatReassigned || evt.Type == eventTypeSeatReassignedLegacy {
		var payload SeatReassignPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
		state.UserID = payload.UserID
	}
	return state
}
