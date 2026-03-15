package session

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// FoldHandledTypes returns the event types handled by the session fold function.
func FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeStarted,
		EventTypeEnded,
		EventTypeGateOpened,
		EventTypeGateResponseRecorded,
		EventTypeGateResolved,
		EventTypeGateAbandoned,
		EventTypeSpotlightSet,
		EventTypeSpotlightCleared,
		EventTypeActiveSceneSet,
		EventTypeGMAuthoritySet,
		EventTypeOOCPaused,
		EventTypeOOCPosted,
		EventTypeOOCReadyMarked,
		EventTypeOOCReadyCleared,
		EventTypeOOCResumed,
		EventTypeAITurnQueued,
		EventTypeAITurnRunning,
		EventTypeAITurnFailed,
		EventTypeAITurnCleared,
	}
}

// Fold applies an event to session state. It returns an error if a recognized
// event carries a payload that cannot be unmarshalled.
//
// The fold is intentionally declarative: every session transition is represented as
// an event so tests and replay both observe the same gate and spotlight behavior.
func Fold(state State, evt event.Event) (State, error) {
	switch evt.Type {
	case EventTypeStarted:
		state.Started = true
		state.Ended = false
		var payload StartPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		state.SessionID = ids.SessionID(payload.SessionID)
		state.Name = payload.SessionName
	case EventTypeEnded:
		state.Ended = true
		state.Started = false
		state.ActiveSceneID = ""
		state.GMAuthorityParticipantID = ""
		state.OOCPaused = false
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
	case EventTypeGateOpened:
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
	case EventTypeGateResponseRecorded:
		// Gate response events do not change the gate-open lifecycle state.
	case EventTypeGateResolved, EventTypeGateAbandoned:
		state.GateOpen = false
		state.GateID = ""
		state.GateType = ""
		state.GateMetadataJSON = nil
	case EventTypeSpotlightSet:
		var payload SpotlightSetPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		state.SpotlightType = payload.SpotlightType
		state.SpotlightCharacterID = ids.CharacterID(payload.CharacterID)
	case EventTypeSpotlightCleared:
		state.SpotlightType = ""
		state.SpotlightCharacterID = ""
	case EventTypeActiveSceneSet:
		var payload ActiveSceneSetPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		state.ActiveSceneID = ids.SceneID(payload.ActiveSceneID)
	case EventTypeGMAuthoritySet:
		var payload GMAuthoritySetPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		state.GMAuthorityParticipantID = ids.ParticipantID(payload.ParticipantID)
	case EventTypeOOCPaused:
		state.OOCPaused = true
		state.OOCReadyParticipants = make(map[ids.ParticipantID]bool)
	case EventTypeOOCPosted:
		// OOC posts do not change session gate/authority state.
	case EventTypeOOCReadyMarked:
		var payload OOCReadyMarkedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		if state.OOCReadyParticipants == nil {
			state.OOCReadyParticipants = make(map[ids.ParticipantID]bool)
		}
		state.OOCReadyParticipants[ids.ParticipantID(payload.ParticipantID)] = true
	case EventTypeOOCReadyCleared:
		var payload OOCReadyClearedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		delete(state.OOCReadyParticipants, ids.ParticipantID(payload.ParticipantID))
	case EventTypeOOCResumed:
		state.OOCPaused = false
		state.OOCReadyParticipants = nil
	case EventTypeAITurnQueued:
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
	case EventTypeAITurnRunning:
		var payload AITurnRunningPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		state.AITurnStatus = AITurnStatusRunning
		state.AITurnToken = strings.TrimSpace(payload.TurnToken)
		state.AITurnLastError = ""
	case EventTypeAITurnFailed:
		var payload AITurnFailedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		state.AITurnStatus = AITurnStatusFailed
		state.AITurnToken = strings.TrimSpace(payload.TurnToken)
		state.AITurnLastError = strings.TrimSpace(payload.LastError)
	case EventTypeAITurnCleared:
		state.AITurnStatus = AITurnStatusIdle
		state.AITurnToken = ""
		state.AITurnOwnerParticipantID = ""
		state.AITurnSourceEventType = ""
		state.AITurnSourceSceneID = ""
		state.AITurnSourcePhaseID = ""
		state.AITurnLastError = ""
	}
	// Unknown event types are silently ignored so that replay remains
	// forward-compatible when new events are added before the fold is updated.
	return state, nil
}
