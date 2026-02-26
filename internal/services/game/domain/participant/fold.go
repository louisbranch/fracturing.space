package participant

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// FoldHandledTypes returns the event types handled by the participant fold function.
func FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeJoined,
		EventTypeUpdated,
		EventTypeLeft,
		EventTypeBound,
		EventTypeUnbound,
		EventTypeSeatReassigned,
		EventTypeSeatReassignedLegacy,
	}
}

// Fold applies an event to participant state. It returns an error if a
// recognized event carries a payload that cannot be unmarshalled.
func Fold(state State, evt event.Event) (State, error) {
	switch evt.Type {
	case EventTypeJoined:
		state.Joined = true
		state.Left = false
		var payload JoinPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
		}
		state.ParticipantID = payload.ParticipantID
		state.UserID = payload.UserID
		state.Name = payload.Name
		state.Role = payload.Role
		state.Controller = payload.Controller
		state.CampaignAccess = payload.CampaignAccess
		state.AvatarSetID = payload.AvatarSetID
		state.AvatarAssetID = payload.AvatarAssetID
		state.Pronouns = payload.Pronouns
	case EventTypeUpdated:
		var payload UpdatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
		}
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
			case "avatar_set_id":
				state.AvatarSetID = value
			case "avatar_asset_id":
				state.AvatarAssetID = value
			case "pronouns":
				state.Pronouns = value
			}
		}
	case EventTypeLeft:
		state.Left = true
		state.Joined = false
		var payload LeavePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
		}
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
	case EventTypeBound:
		var payload BindPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
		}
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
		state.UserID = payload.UserID
	case EventTypeUnbound:
		var payload UnbindPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
		}
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
		state.UserID = ""
	case EventTypeSeatReassigned, EventTypeSeatReassignedLegacy:
		var payload SeatReassignPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
		}
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
		state.UserID = payload.UserID
	}
	return state, nil
}
