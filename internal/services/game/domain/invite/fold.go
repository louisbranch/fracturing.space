package invite

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// FoldHandledTypes returns the event types handled by the invite fold function.
func FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeCreated,
		EventTypeClaimed,
		EventTypeRevoked,
		EventTypeUpdated,
	}
}

// Fold applies an event to invite state. It returns an error if a recognized
// event carries a payload that cannot be unmarshalled.
func Fold(state State, evt event.Event) (State, error) {
	switch evt.Type {
	case EventTypeCreated:
		state.Created = true
		var payload CreatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("invite fold %s: %w", evt.Type, err)
		}
		state.InviteID = payload.InviteID
		state.ParticipantID = payload.ParticipantID
		state.RecipientUserID = payload.RecipientUserID
		state.CreatedByParticipantID = payload.CreatedByParticipantID
		status := payload.Status
		if normalized, ok := normalizeStatusLabel(payload.Status); ok {
			status = normalized
		}
		state.Status = status
	case EventTypeClaimed:
		var payload ClaimPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("invite fold %s: %w", evt.Type, err)
		}
		if payload.InviteID != "" {
			state.InviteID = payload.InviteID
		}
		if payload.ParticipantID != "" {
			state.ParticipantID = payload.ParticipantID
		}
		state.Status = statusClaimed
	case EventTypeRevoked:
		var payload RevokePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("invite fold %s: %w", evt.Type, err)
		}
		if payload.InviteID != "" {
			state.InviteID = payload.InviteID
		}
		state.Status = statusRevoked
	case EventTypeUpdated:
		var payload UpdatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("invite fold %s: %w", evt.Type, err)
		}
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
	return state, nil
}
