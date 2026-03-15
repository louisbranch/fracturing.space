package scene

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
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
		EventTypePlayerPhaseStarted,
		EventTypePlayerPhasePosted,
		EventTypePlayerPhaseYielded,
		EventTypePlayerPhaseReviewStarted,
		EventTypePlayerPhaseUnyielded,
		EventTypePlayerPhaseRevisionsRequested,
		EventTypePlayerPhaseAccepted,
		EventTypePlayerPhaseEnded,
		EventTypeGMOutputCommitted,
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
		state.SceneID = ids.SceneID(payload.SceneID)
		state.Name = payload.Name
		state.Description = payload.Description
		state.Active = true
		if state.Characters == nil {
			state.Characters = make(map[ids.CharacterID]bool)
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
		state.PlayerPhaseID = ""
		state.PlayerPhaseFrameText = ""
		state.PlayerPhaseStatus = ""
		state.PlayerPhaseActingCharacters = nil
		state.PlayerPhaseActingParticipants = nil
		state.PlayerPhaseSlots = nil

	case EventTypeCharacterAdded:
		var payload CharacterAddedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		if state.Characters == nil {
			state.Characters = make(map[ids.CharacterID]bool)
		}
		state.Characters[ids.CharacterID(payload.CharacterID)] = true

	case EventTypeCharacterRemoved:
		var payload CharacterRemovedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		delete(state.Characters, ids.CharacterID(payload.CharacterID))

	case EventTypeGateOpened:
		var payload GateOpenedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		state.GateOpen = true
		state.GateID = ids.GateID(payload.GateID)

	case EventTypeGateResolved, EventTypeGateAbandoned:
		state.GateOpen = false
		state.GateID = ""

	case EventTypeSpotlightSet:
		var payload SpotlightSetPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		state.SpotlightType = payload.SpotlightType
		state.SpotlightCharacterID = ids.CharacterID(payload.CharacterID)

	case EventTypeSpotlightCleared:
		state.SpotlightType = ""
		state.SpotlightCharacterID = ""
	case EventTypePlayerPhaseStarted:
		var payload PlayerPhaseStartedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		state.PlayerPhaseID = payload.PhaseID
		state.PlayerPhaseFrameText = payload.FrameText
		state.PlayerPhaseStatus = PlayerPhaseStatusPlayers
		state.PlayerPhaseActingCharacters = append([]ids.CharacterID(nil), payload.ActingCharacterIDs...)
		state.PlayerPhaseActingParticipants = make(map[ids.ParticipantID]bool, len(payload.ActingParticipantIDs))
		for _, participantID := range payload.ActingParticipantIDs {
			state.PlayerPhaseActingParticipants[ids.ParticipantID(participantID)] = true
		}
		state.PlayerPhaseSlots = make(map[ids.ParticipantID]PlayerPhaseSlot, len(payload.ActingParticipantIDs))
		for _, participantID := range payload.ActingParticipantIDs {
			state.PlayerPhaseSlots[participantID] = PlayerPhaseSlot{
				ParticipantID: participantID,
				Yielded:       false,
				ReviewStatus:  PlayerPhaseSlotReviewStatusOpen,
			}
		}
	case EventTypePlayerPhasePosted:
		var payload PlayerPhasePostedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		if state.PlayerPhaseSlots == nil {
			state.PlayerPhaseSlots = make(map[ids.ParticipantID]PlayerPhaseSlot)
		}
		participantID := ids.ParticipantID(payload.ParticipantID)
		slot := state.PlayerPhaseSlots[participantID]
		slot.ParticipantID = participantID
		slot.CharacterIDs = append([]ids.CharacterID(nil), payload.CharacterIDs...)
		slot.SummaryText = payload.SummaryText
		slot.Yielded = false
		slot.ReviewStatus = PlayerPhaseSlotReviewStatusOpen
		slot.ReviewReason = ""
		slot.ReviewCharacterIDs = nil
		state.PlayerPhaseSlots[participantID] = slot
	case EventTypePlayerPhaseYielded:
		var payload PlayerPhaseYieldedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		if state.PlayerPhaseSlots == nil {
			state.PlayerPhaseSlots = make(map[ids.ParticipantID]PlayerPhaseSlot)
		}
		participantID := ids.ParticipantID(payload.ParticipantID)
		slot := state.PlayerPhaseSlots[participantID]
		slot.ParticipantID = participantID
		slot.Yielded = true
		state.PlayerPhaseSlots[participantID] = slot
	case EventTypePlayerPhaseReviewStarted:
		state.PlayerPhaseStatus = PlayerPhaseStatusGMReview
		for participantID, slot := range state.PlayerPhaseSlots {
			slot.ReviewStatus = PlayerPhaseSlotReviewStatusUnderReview
			slot.ReviewReason = ""
			slot.ReviewCharacterIDs = nil
			state.PlayerPhaseSlots[participantID] = slot
		}
	case EventTypePlayerPhaseUnyielded:
		var payload PlayerPhaseUnyieldedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		participantID := ids.ParticipantID(payload.ParticipantID)
		slot := state.PlayerPhaseSlots[participantID]
		slot.ParticipantID = participantID
		slot.Yielded = false
		slot.ReviewStatus = PlayerPhaseSlotReviewStatusOpen
		slot.ReviewReason = ""
		slot.ReviewCharacterIDs = nil
		state.PlayerPhaseSlots[participantID] = slot
	case EventTypePlayerPhaseRevisionsRequested:
		var payload PlayerPhaseRevisionsRequestedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		state.PlayerPhaseStatus = PlayerPhaseStatusPlayers
		targeted := make(map[ids.ParticipantID]PlayerPhaseRevisionRequest, len(payload.Revisions))
		for _, revision := range payload.Revisions {
			targeted[revision.ParticipantID] = revision
		}
		for participantID, slot := range state.PlayerPhaseSlots {
			if revision, ok := targeted[participantID]; ok {
				slot.Yielded = false
				slot.ReviewStatus = PlayerPhaseSlotReviewStatusChangesRequested
				slot.ReviewReason = revision.Reason
				slot.ReviewCharacterIDs = append([]ids.CharacterID(nil), revision.CharacterIDs...)
			} else {
				slot.ReviewStatus = PlayerPhaseSlotReviewStatusAccepted
				slot.ReviewReason = ""
				slot.ReviewCharacterIDs = nil
			}
			state.PlayerPhaseSlots[participantID] = slot
		}
	case EventTypePlayerPhaseAccepted:
		for participantID, slot := range state.PlayerPhaseSlots {
			slot.ReviewStatus = PlayerPhaseSlotReviewStatusAccepted
			slot.ReviewReason = ""
			slot.ReviewCharacterIDs = nil
			state.PlayerPhaseSlots[participantID] = slot
		}
	case EventTypePlayerPhaseEnded:
		state.PlayerPhaseID = ""
		state.PlayerPhaseFrameText = ""
		state.PlayerPhaseStatus = ""
		state.PlayerPhaseActingCharacters = nil
		state.PlayerPhaseActingParticipants = nil
		state.PlayerPhaseSlots = nil
	case EventTypeGMOutputCommitted:
		var payload GMOutputCommittedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
		}
		state.GMOutputText = payload.Text
		state.GMOutputParticipantID = payload.ParticipantID
	}
	// Unknown event types are silently ignored so that replay remains
	// forward-compatible when new events are added before the fold is updated.
	return state, nil
}
