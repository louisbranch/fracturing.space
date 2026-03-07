package scene

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// FoldHandledTypes returns the event types handled by the scene fold function.
func FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeCreated,
		EventTypeUpdated,
		EventTypeEnded,
		EventTypeCharacterAdded,
		EventTypeCharacterRemoved,
		EventTypeGateOpened,
		EventTypeGateResolved,
		EventTypeGateAbandoned,
		EventTypeSpotlightSet,
		EventTypeSpotlightCleared,
	}
}

// Fold applies an event to scene state within the scenes map. It handles
// entity-keyed scene state by looking up the target scene from the event's
// EntityID (for most events) or from the event payload's SceneID field.
//
// The fold is deterministic: same events produce the same scene states.
func Fold(state State, evt event.Event) (State, error) {
	switch evt.Type {
	case EventTypeCreated:
		var payload CreatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		state.SceneID = payload.SceneID
		state.Name = payload.Name
		state.Description = payload.Description
		state.Active = true
		if state.Characters == nil {
			state.Characters = make(map[string]bool)
		}

	case EventTypeUpdated:
		var payload UpdatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		if payload.Name != "" {
			state.Name = payload.Name
		}
		if payload.Description != "" {
			state.Description = payload.Description
		}

	case EventTypeEnded:
		state.Active = false
		state.GateOpen = false
		state.GateID = ""
		state.SpotlightType = ""
		state.SpotlightCharacterID = ""

	case EventTypeCharacterAdded:
		var payload CharacterAddedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		if state.Characters == nil {
			state.Characters = make(map[string]bool)
		}
		state.Characters[payload.CharacterID] = true

	case EventTypeCharacterRemoved:
		var payload CharacterRemovedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		delete(state.Characters, payload.CharacterID)

	case EventTypeGateOpened:
		var payload GateOpenedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		state.GateOpen = true
		state.GateID = payload.GateID

	case EventTypeGateResolved, EventTypeGateAbandoned:
		state.GateOpen = false
		state.GateID = ""

	case EventTypeSpotlightSet:
		var payload SpotlightSetPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		state.SpotlightType = payload.SpotlightType
		state.SpotlightCharacterID = payload.CharacterID

	case EventTypeSpotlightCleared:
		state.SpotlightType = ""
		state.SpotlightCharacterID = ""
	}

	return state, nil
}
