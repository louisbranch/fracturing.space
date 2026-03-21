package invite

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fold"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// foldRouter is the registration-based fold dispatcher. Handled types are
// derived from registered handlers, eliminating sync-drift between the switch
// and the type list.
var foldRouter = newFoldRouter()

func newFoldRouter() *fold.CoreFoldRouter[State] {
	r := fold.NewCoreFoldRouter[State]()
	r.Handle(EventTypeCreated, foldCreated)
	r.Handle(EventTypeClaimed, foldClaimed)
	r.Handle(EventTypeDeclined, foldDeclined)
	r.Handle(EventTypeRevoked, foldRevoked)
	r.Handle(EventTypeUpdated, foldUpdated)
	return r
}

// FoldHandledTypes returns the event types handled by the invite fold function.
// Derived from registered handlers via the fold router.
func FoldHandledTypes() []event.Type {
	return foldRouter.FoldHandledTypes()
}

// Fold applies an event to invite state. Returns an error for unhandled
// event types and for recognized events with unparseable payloads.
func Fold(state State, evt event.Event) (State, error) {
	return foldRouter.Fold(state, evt)
}

func foldCreated(state State, evt event.Event) (State, error) {
	state.Created = true
	var payload CreatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("invite fold %s: %w", evt.Type, err)
	}
	state.InviteID = ids.InviteID(payload.InviteID)
	state.ParticipantID = ids.ParticipantID(payload.ParticipantID)
	state.RecipientUserID = ids.UserID(payload.RecipientUserID)
	state.CreatedByParticipantID = ids.ParticipantID(payload.CreatedByParticipantID)
	if normalized, ok := NormalizeStatus(payload.Status); ok {
		state.Status = normalized
	} else {
		state.Status = Status(payload.Status)
	}
	return state, nil
}

func foldClaimed(state State, evt event.Event) (State, error) {
	var payload ClaimPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("invite fold %s: %w", evt.Type, err)
	}
	if payload.InviteID != "" {
		state.InviteID = ids.InviteID(payload.InviteID)
	}
	if payload.ParticipantID != "" {
		state.ParticipantID = ids.ParticipantID(payload.ParticipantID)
	}
	state.Status = statusClaimed
	return state, nil
}

func foldDeclined(state State, evt event.Event) (State, error) {
	var payload DeclinePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("invite fold %s: %w", evt.Type, err)
	}
	if payload.InviteID != "" {
		state.InviteID = ids.InviteID(payload.InviteID)
	}
	state.Status = statusDeclined
	return state, nil
}

func foldRevoked(state State, evt event.Event) (State, error) {
	var payload RevokePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("invite fold %s: %w", evt.Type, err)
	}
	if payload.InviteID != "" {
		state.InviteID = ids.InviteID(payload.InviteID)
	}
	state.Status = statusRevoked
	return state, nil
}

func foldUpdated(state State, evt event.Event) (State, error) {
	var payload UpdatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("invite fold %s: %w", evt.Type, err)
	}
	if payload.InviteID != "" {
		state.InviteID = ids.InviteID(payload.InviteID)
	}
	if payload.Status != "" {
		if normalized, ok := NormalizeStatus(payload.Status); ok {
			state.Status = normalized
		} else {
			state.Status = Status(payload.Status)
		}
	}
	return state, nil
}
