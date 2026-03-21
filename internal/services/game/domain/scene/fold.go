package scene

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
	r.Handle(EventTypeUpdated, foldUpdated)
	r.Handle(EventTypeEnded, foldEnded)
	r.Handle(EventTypeCharacterAdded, foldCharacterAdded)
	r.Handle(EventTypeCharacterRemoved, foldCharacterRemoved)
	r.Handle(EventTypeGateOpened, foldGateOpened)
	r.Handle(EventTypeGateResolved, foldGateClosed)
	r.Handle(EventTypeGateAbandoned, foldGateClosed)
	r.Handle(EventTypeSpotlightSet, foldSpotlightSet)
	r.Handle(EventTypeSpotlightCleared, foldSpotlightCleared)
	r.Handle(EventTypePlayerPhaseStarted, foldPlayerPhaseStarted)
	r.Handle(EventTypePlayerPhasePosted, foldPlayerPhasePosted)
	r.Handle(EventTypePlayerPhaseYielded, foldPlayerPhaseYielded)
	r.Handle(EventTypePlayerPhaseReviewStarted, foldPlayerPhaseReviewStarted)
	r.Handle(EventTypePlayerPhaseUnyielded, foldPlayerPhaseUnyielded)
	r.Handle(EventTypePlayerPhaseRevisionsRequested, foldPlayerPhaseRevisionsRequested)
	r.Handle(EventTypePlayerPhaseAccepted, foldPlayerPhaseAccepted)
	r.Handle(EventTypePlayerPhaseEnded, foldPlayerPhaseEnded)
	r.Handle(EventTypeGMOutputCommitted, foldGMOutputCommitted)
	return r
}

// FoldHandledTypes returns the event types handled by the scene fold function.
// Derived from registered handlers via the fold router.
func FoldHandledTypes() []event.Type {
	return foldRouter.FoldHandledTypes()
}

// Fold applies an event to scene state within the scenes map. It handles
// entity-keyed scene state by looking up the target scene from the event's
// EntityID (for most events) or from the event payload's SceneID field.
//
// The fold is deterministic: same events produce the same scene states.
func Fold(state State, evt event.Event) (State, error) {
	return foldRouter.Fold(state, evt)
}

func foldCreated(state State, evt event.Event) (State, error) {
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
	return state, nil
}

func foldUpdated(state State, evt event.Event) (State, error) {
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
	return state, nil
}

func foldEnded(state State, _ event.Event) (State, error) {
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
	return state, nil
}

func foldCharacterAdded(state State, evt event.Event) (State, error) {
	var payload CharacterAddedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
	}
	if state.Characters == nil {
		state.Characters = make(map[ids.CharacterID]bool)
	}
	state.Characters[ids.CharacterID(payload.CharacterID)] = true
	return state, nil
}

func foldCharacterRemoved(state State, evt event.Event) (State, error) {
	var payload CharacterRemovedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
	}
	delete(state.Characters, ids.CharacterID(payload.CharacterID))
	return state, nil
}

func foldGateOpened(state State, evt event.Event) (State, error) {
	var payload GateOpenedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
	}
	state.GateOpen = true
	state.GateID = ids.GateID(payload.GateID)
	return state, nil
}

// foldGateClosed handles both gate.resolved and gate.abandoned.
func foldGateClosed(state State, _ event.Event) (State, error) {
	state.GateOpen = false
	state.GateID = ""
	return state, nil
}

func foldSpotlightSet(state State, evt event.Event) (State, error) {
	var payload SpotlightSetPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
	}
	state.SpotlightType = payload.SpotlightType
	state.SpotlightCharacterID = ids.CharacterID(payload.CharacterID)
	return state, nil
}

func foldSpotlightCleared(state State, _ event.Event) (State, error) {
	state.SpotlightType = ""
	state.SpotlightCharacterID = ""
	return state, nil
}

func foldPlayerPhaseStarted(state State, evt event.Event) (State, error) {
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
	return state, nil
}

func foldPlayerPhasePosted(state State, evt event.Event) (State, error) {
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
	return state, nil
}

func foldPlayerPhaseYielded(state State, evt event.Event) (State, error) {
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
	return state, nil
}

func foldPlayerPhaseReviewStarted(state State, _ event.Event) (State, error) {
	state.PlayerPhaseStatus = PlayerPhaseStatusGMReview
	for participantID, slot := range state.PlayerPhaseSlots {
		slot.ReviewStatus = PlayerPhaseSlotReviewStatusUnderReview
		slot.ReviewReason = ""
		slot.ReviewCharacterIDs = nil
		state.PlayerPhaseSlots[participantID] = slot
	}
	return state, nil
}

func foldPlayerPhaseUnyielded(state State, evt event.Event) (State, error) {
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
	return state, nil
}

func foldPlayerPhaseRevisionsRequested(state State, evt event.Event) (State, error) {
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
	return state, nil
}

func foldPlayerPhaseAccepted(state State, _ event.Event) (State, error) {
	for participantID, slot := range state.PlayerPhaseSlots {
		slot.ReviewStatus = PlayerPhaseSlotReviewStatusAccepted
		slot.ReviewReason = ""
		slot.ReviewCharacterIDs = nil
		state.PlayerPhaseSlots[participantID] = slot
	}
	return state, nil
}

func foldPlayerPhaseEnded(state State, _ event.Event) (State, error) {
	state.PlayerPhaseID = ""
	state.PlayerPhaseFrameText = ""
	state.PlayerPhaseStatus = ""
	state.PlayerPhaseActingCharacters = nil
	state.PlayerPhaseActingParticipants = nil
	state.PlayerPhaseSlots = nil
	return state, nil
}

func foldGMOutputCommitted(state State, evt event.Event) (State, error) {
	var payload GMOutputCommittedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("scene fold %s: %w", evt.Type, err)
	}
	state.GMOutputText = payload.Text
	state.GMOutputParticipantID = payload.ParticipantID
	return state, nil
}
