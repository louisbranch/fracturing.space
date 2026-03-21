package participant

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fold"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

type participantUpdateFieldApplier func(*State, string) error

var participantUpdateFieldAppliers = map[string]participantUpdateFieldApplier{
	"user_id": func(state *State, value string) error {
		state.UserID = ids.UserID(value)
		return nil
	},
	"name": func(state *State, value string) error {
		state.Name = value
		return nil
	},
	"role": func(state *State, value string) error {
		state.Role = Role(value)
		return nil
	},
	"controller": func(state *State, value string) error {
		state.Controller = Controller(value)
		return nil
	},
	"campaign_access": func(state *State, value string) error {
		state.CampaignAccess = CampaignAccess(value)
		return nil
	},
	"avatar_set_id": func(state *State, value string) error {
		state.AvatarSetID = value
		return nil
	},
	"avatar_asset_id": func(state *State, value string) error {
		state.AvatarAssetID = value
		return nil
	},
	"pronouns": func(state *State, value string) error {
		state.Pronouns = value
		return nil
	},
}

// foldRouter is the registration-based fold dispatcher. Handled types are
// derived from registered handlers, eliminating sync-drift between the switch
// and the type list.
var foldRouter = newFoldRouter()

func newFoldRouter() *fold.CoreFoldRouter[State] {
	r := fold.NewCoreFoldRouter[State]()
	r.Handle(EventTypeJoined, foldJoined)
	r.Handle(EventTypeUpdated, foldUpdated)
	r.Handle(EventTypeLeft, foldLeft)
	r.Handle(EventTypeBound, foldBound)
	r.Handle(EventTypeUnbound, foldUnbound)
	r.Handle(EventTypeSeatReassigned, foldSeatReassigned)
	return r
}

// FoldHandledTypes returns the event types handled by the participant fold function.
// Derived from registered handlers via the fold router.
func FoldHandledTypes() []event.Type {
	return foldRouter.FoldHandledTypes()
}

// Fold applies an event to participant state. Returns an error for unhandled
// event types and for recognized events with unparseable payloads.
func Fold(state State, evt event.Event) (State, error) {
	return foldRouter.Fold(state, evt)
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
	if err := applyParticipantUpdateFields(&state, payload.Fields); err != nil {
		return state, fmt.Errorf("participant fold %s: %w", evt.Type, err)
	}
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

func applyParticipantUpdateFields(state *State, fields map[string]string) error {
	for key, value := range fields {
		applier, ok := participantUpdateFieldAppliers[key]
		if !ok {
			continue
		}
		if err := applier(state, value); err != nil {
			return err
		}
	}
	return nil
}
