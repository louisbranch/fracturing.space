package participant

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

type participantUpdateFieldApplier func(*State, string)

var participantUpdateFieldAppliers = map[string]participantUpdateFieldApplier{
	"user_id": func(state *State, value string) {
		state.UserID = ids.UserID(value)
	},
	"name": func(state *State, value string) {
		state.Name = value
	},
	"role": func(state *State, value string) {
		state.Role = Role(value)
	},
	"controller": func(state *State, value string) {
		state.Controller = Controller(value)
	},
	"campaign_access": func(state *State, value string) {
		state.CampaignAccess = CampaignAccess(value)
	},
	"avatar_set_id": func(state *State, value string) {
		state.AvatarSetID = value
	},
	"avatar_asset_id": func(state *State, value string) {
		state.AvatarAssetID = value
	},
	"pronouns": func(state *State, value string) {
		state.Pronouns = value
	},
}

// FoldHandledTypes returns the event types handled by the participant fold function.
func FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeJoined,
		EventTypeUpdated,
		EventTypeLeft,
		EventTypeBound,
		EventTypeUnbound,
		EventTypeSeatReassigned,
	}
}

// Fold applies an event to participant state. It returns an error if a
// recognized event carries a payload that cannot be unmarshalled.
func Fold(state State, evt event.Event) (State, error) {
	switch evt.Type {
	case EventTypeJoined:
		return foldJoined(state, evt)
	case EventTypeUpdated:
		return foldUpdated(state, evt)
	case EventTypeLeft:
		return foldLeft(state, evt)
	case EventTypeBound:
		return foldBound(state, evt)
	case EventTypeUnbound:
		return foldUnbound(state, evt)
	case EventTypeSeatReassigned:
		return foldSeatReassigned(state, evt)
	}
	// Unknown event types are silently ignored so that replay remains
	// forward-compatible when new events are added before the fold is updated.
	return state, nil
}

func foldJoined(state State, evt event.Event) (State, error) {
	state.Joined = true
	state.Left = false

	var payload JoinPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
	}
	state.ParticipantID = ids.ParticipantID(payload.ParticipantID)
	state.UserID = ids.UserID(payload.UserID)
	state.Name = payload.Name
	state.Role = Role(payload.Role)
	state.Controller = Controller(payload.Controller)
	state.CampaignAccess = CampaignAccess(payload.CampaignAccess)
	state.AvatarSetID = payload.AvatarSetID
	state.AvatarAssetID = payload.AvatarAssetID
	state.Pronouns = payload.Pronouns
	return state, nil
}

func foldUpdated(state State, evt event.Event) (State, error) {
	var payload UpdatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
	}
	if payload.ParticipantID != "" {
		state.ParticipantID = ids.ParticipantID(payload.ParticipantID)
	}
	applyParticipantUpdateFields(&state, payload.Fields)
	return state, nil
}

func foldLeft(state State, evt event.Event) (State, error) {
	state.Left = true
	state.Joined = false

	var payload LeavePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
	}
	if payload.ParticipantID != "" {
		state.ParticipantID = ids.ParticipantID(payload.ParticipantID)
	}
	return state, nil
}

func foldBound(state State, evt event.Event) (State, error) {
	var payload BindPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
	}
	if payload.ParticipantID != "" {
		state.ParticipantID = ids.ParticipantID(payload.ParticipantID)
	}
	state.UserID = ids.UserID(payload.UserID)
	return state, nil
}

func foldUnbound(state State, evt event.Event) (State, error) {
	var payload UnbindPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
	}
	if payload.ParticipantID != "" {
		state.ParticipantID = ids.ParticipantID(payload.ParticipantID)
	}
	state.UserID = ""
	return state, nil
}

func foldSeatReassigned(state State, evt event.Event) (State, error) {
	var payload SeatReassignPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
	}
	if payload.ParticipantID != "" {
		state.ParticipantID = ids.ParticipantID(payload.ParticipantID)
	}
	state.UserID = ids.UserID(payload.UserID)
	return state, nil
}

func applyParticipantUpdateFields(state *State, fields map[string]string) {
	for key, value := range fields {
		applier, ok := participantUpdateFieldAppliers[key]
		if !ok {
			continue
		}
		applier(state, value)
	}
}
