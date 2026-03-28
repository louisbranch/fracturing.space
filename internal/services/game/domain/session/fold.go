package session

import (
	"encoding/json"
	"fmt"
	"strings"

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
	r.Handle(EventTypeStarted, foldStarted)
	r.Handle(EventTypeEnded, foldEnded)
	r.Handle(EventTypeGateOpened, foldGateOpened)
	r.Handle(EventTypeGateResponseRecorded, foldGateResponseRecorded)
	r.Handle(EventTypeGateResolved, foldGateClosed)
	r.Handle(EventTypeGateAbandoned, foldGateClosed)
	r.Handle(EventTypeSpotlightSet, foldSpotlightSet)
	r.Handle(EventTypeSpotlightCleared, foldSpotlightCleared)
	r.Handle(EventTypeSceneActivated, foldSceneActivated)
	r.Handle(EventTypeGMAuthoritySet, foldGMAuthoritySet)
	r.Handle(EventTypeCharacterControllerSet, foldCharacterControllerSet)
	r.Handle(EventTypeOOCOpened, foldOOCOpened)
	r.Handle(EventTypeOOCPosted, foldOOCPosted)
	r.Handle(EventTypeOOCReadyMarked, foldOOCReadyMarked)
	r.Handle(EventTypeOOCReadyCleared, foldOOCReadyCleared)
	r.Handle(EventTypeOOCClosed, foldOOCClosed)
	r.Handle(EventTypeOOCResolved, foldOOCResolved)
	r.Handle(EventTypeAITurnQueued, foldAITurnQueued)
	r.Handle(EventTypeAITurnRunning, foldAITurnRunning)
	r.Handle(EventTypeAITurnFailed, foldAITurnFailed)
	r.Handle(EventTypeAITurnCleared, foldAITurnCleared)
	return r
}

// FoldHandledTypes returns the event types handled by the session fold function.
// Derived from registered handlers via the fold router.
func FoldHandledTypes() []event.Type {
	return foldRouter.FoldHandledTypes()
}

// Fold applies an event to session state. Returns an error for unhandled
// event types and for recognized events with unparseable payloads.
//
// The fold is intentionally declarative: every session transition is represented as
// an event so tests and replay both observe the same gate and spotlight behavior.
func Fold(state State, evt event.Event) (State, error) {
	return foldRouter.Fold(state, evt)
}

func foldStarted(state State, evt event.Event) (State, error) {
	state.Started = true
	state.Ended = false
	var payload StartPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	state.SessionID = ids.SessionID(payload.SessionID)
	state.Name = payload.SessionName
	state.CharacterControllers = make(map[ids.CharacterID]ids.ParticipantID, len(payload.CharacterControllers))
	for _, assignment := range payload.CharacterControllers {
		if assignment.CharacterID == "" || assignment.ParticipantID == "" {
			continue
		}
		state.CharacterControllers[assignment.CharacterID] = assignment.ParticipantID
	}
	return state, nil
}

func foldEnded(state State, evt event.Event) (State, error) {
	state.Ended = true
	state.Started = false
	state.CharacterControllers = nil
	state.ActiveSceneID = ""
	state.GMAuthorityParticipantID = ""
	state.OOCPaused = false
	state.OOCRequestedByParticipantID = ""
	state.OOCReason = ""
	state.OOCInterruptedSceneID = ""
	state.OOCInterruptedPhaseID = ""
	state.OOCInterruptedPhaseStatus = ""
	state.OOCResolutionPending = false
	state.OOCReadyParticipants = nil
	state.AITurnStatus = ""
	state.AITurnToken = ""
	state.AITurnOwnerParticipantID = ""
	state.AITurnSourceEventType = ""
	state.AITurnSourceSceneID = ""
	state.AITurnSourcePhaseID = ""
	state.AITurnLastError = ""
	var payload EndPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	if payload.SessionID != "" {
		state.SessionID = ids.SessionID(payload.SessionID)
	}
	return state, nil
}

func foldGateOpened(state State, evt event.Event) (State, error) {
	state.GateOpen = true
	var payload GateOpenedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	state.GateID = ids.GateID(payload.GateID)
	state.GateType = strings.TrimSpace(payload.GateType)
	metadataJSON, err := MarshalGateMetadataJSON(state.GateType, payload.Metadata)
	if err != nil {
		return state, fmt.Errorf("session fold %s metadata: %w", evt.Type, err)
	}
	state.GateMetadataJSON = metadataJSON
	return state, nil
}

// foldGateResponseRecorded is a no-op: gate response events do not change the
// gate-open lifecycle state.
func foldGateResponseRecorded(state State, _ event.Event) (State, error) {
	return state, nil
}

// foldGateClosed handles both gate.resolved and gate.abandoned.
func foldGateClosed(state State, _ event.Event) (State, error) {
	state.GateOpen = false
	state.GateID = ""
	state.GateType = ""
	state.GateMetadataJSON = nil
	return state, nil
}

func foldSpotlightSet(state State, evt event.Event) (State, error) {
	var payload SpotlightSetPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
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

func foldSceneActivated(state State, evt event.Event) (State, error) {
	var payload SceneActivatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	state.ActiveSceneID = ids.SceneID(payload.ActiveSceneID)
	return state, nil
}

func foldGMAuthoritySet(state State, evt event.Event) (State, error) {
	var payload GMAuthoritySetPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	state.GMAuthorityParticipantID = ids.ParticipantID(payload.ParticipantID)
	return state, nil
}

func foldCharacterControllerSet(state State, evt event.Event) (State, error) {
	var payload CharacterControllerSetPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	if state.CharacterControllers == nil {
		state.CharacterControllers = make(map[ids.CharacterID]ids.ParticipantID)
	}
	state.CharacterControllers[payload.CharacterID] = payload.ParticipantID
	return state, nil
}

func foldOOCOpened(state State, evt event.Event) (State, error) {
	var payload OOCOpenedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	state.OOCPaused = true
	state.OOCRequestedByParticipantID = ids.ParticipantID(payload.RequestedByParticipantID)
	state.OOCReason = strings.TrimSpace(payload.Reason)
	state.OOCInterruptedSceneID = ids.SceneID(payload.InterruptedSceneID)
	state.OOCInterruptedPhaseID = strings.TrimSpace(payload.InterruptedPhaseID)
	state.OOCInterruptedPhaseStatus = strings.TrimSpace(payload.InterruptedPhaseStatus)
	state.OOCResolutionPending = false
	state.OOCReadyParticipants = make(map[ids.ParticipantID]bool)
	return state, nil
}

// foldOOCPosted is a no-op: OOC posts do not change session gate/authority state.
func foldOOCPosted(state State, _ event.Event) (State, error) {
	return state, nil
}

func foldOOCReadyMarked(state State, evt event.Event) (State, error) {
	var payload OOCReadyMarkedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	if state.OOCReadyParticipants == nil {
		state.OOCReadyParticipants = make(map[ids.ParticipantID]bool)
	}
	state.OOCReadyParticipants[ids.ParticipantID(payload.ParticipantID)] = true
	return state, nil
}

func foldOOCReadyCleared(state State, evt event.Event) (State, error) {
	var payload OOCReadyClearedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	delete(state.OOCReadyParticipants, ids.ParticipantID(payload.ParticipantID))
	return state, nil
}

func foldOOCClosed(state State, _ event.Event) (State, error) {
	state.OOCPaused = false
	state.OOCReadyParticipants = nil
	state.OOCResolutionPending = state.OOCInterruptedSceneID != "" && state.OOCInterruptedPhaseID != ""
	return state, nil
}

func foldOOCResolved(state State, _ event.Event) (State, error) {
	state.OOCRequestedByParticipantID = ""
	state.OOCReason = ""
	state.OOCInterruptedSceneID = ""
	state.OOCInterruptedPhaseID = ""
	state.OOCInterruptedPhaseStatus = ""
	state.OOCResolutionPending = false
	return state, nil
}

func foldAITurnQueued(state State, evt event.Event) (State, error) {
	var payload AITurnQueuedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	state.AITurnStatus = AITurnStatusQueued
	state.AITurnToken = strings.TrimSpace(payload.TurnToken)
	state.AITurnOwnerParticipantID = ids.ParticipantID(payload.OwnerParticipantID)
	state.AITurnSourceEventType = strings.TrimSpace(payload.SourceEventType)
	state.AITurnSourceSceneID = ids.SceneID(payload.SourceSceneID)
	state.AITurnSourcePhaseID = strings.TrimSpace(payload.SourcePhaseID)
	state.AITurnLastError = ""
	return state, nil
}

func foldAITurnRunning(state State, evt event.Event) (State, error) {
	var payload AITurnRunningPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	state.AITurnStatus = AITurnStatusRunning
	state.AITurnToken = strings.TrimSpace(payload.TurnToken)
	state.AITurnLastError = ""
	return state, nil
}

func foldAITurnFailed(state State, evt event.Event) (State, error) {
	var payload AITurnFailedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
	}
	state.AITurnStatus = AITurnStatusFailed
	state.AITurnToken = strings.TrimSpace(payload.TurnToken)
	state.AITurnLastError = strings.TrimSpace(payload.LastError)
	return state, nil
}

func foldAITurnCleared(state State, _ event.Event) (State, error) {
	state.AITurnStatus = AITurnStatusIdle
	state.AITurnToken = ""
	state.AITurnOwnerParticipantID = ""
	state.AITurnSourceEventType = ""
	state.AITurnSourceSceneID = ""
	state.AITurnSourcePhaseID = ""
	state.AITurnLastError = ""
	return state, nil
}
