package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeStart          command.Type = "session.start"
	commandTypeEnd            command.Type = "session.end"
	commandTypeGateOpen       command.Type = "session.gate_open"
	commandTypeGateResolve    command.Type = "session.gate_resolve"
	commandTypeGateAbandon    command.Type = "session.gate_abandon"
	commandTypeSpotlightSet   command.Type = "session.spotlight_set"
	commandTypeSpotlightClear command.Type = "session.spotlight_clear"
	EventTypeStarted          event.Type   = "session.started"
	EventTypeEnded            event.Type   = "session.ended"
	EventTypeGateOpened       event.Type   = "session.gate_opened"
	EventTypeGateResolved     event.Type   = "session.gate_resolved"
	EventTypeGateAbandoned    event.Type   = "session.gate_abandoned"
	EventTypeSpotlightSet     event.Type   = "session.spotlight_set"
	EventTypeSpotlightCleared event.Type   = "session.spotlight_cleared"

	rejectionCodeSessionIDRequired            = "SESSION_ID_REQUIRED"
	rejectionCodeSessionAlreadyStarted        = "SESSION_ALREADY_STARTED"
	rejectionCodeSessionNotStarted            = "SESSION_NOT_STARTED"
	rejectionCodeSessionGateIDRequired        = "SESSION_GATE_ID_REQUIRED"
	rejectionCodeSessionGateTypeRequired      = "SESSION_GATE_TYPE_REQUIRED"
	rejectionCodeSessionSpotlightTypeRequired = "SESSION_SPOTLIGHT_TYPE_REQUIRED"
)

// Decide returns the decision for a session command against current state.
//
// It maps every supported session lifecycle and gate command to deterministic
// events, and leaves status checks to replayable state transitions rather than
// imperative side effects.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	switch cmd.Type {
	case commandTypeStart:
		if state.Started {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionAlreadyStarted,
				Message: "session already started",
			})
		}
		var payload StartPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: "PAYLOAD_DECODE_FAILED", Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		sessionID := strings.TrimSpace(payload.SessionID)
		if sessionID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionIDRequired,
				Message: "session id is required",
			})
		}
		sessionName := strings.TrimSpace(payload.SessionName)
		if now == nil {
			now = time.Now
		}

		normalizedPayload := StartPayload{SessionID: sessionID, SessionName: sessionName}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeStarted, "session", sessionID, payloadJSON, now().UTC())
		evt.SessionID = sessionID

		return command.Accept(evt)

	case commandTypeEnd:
		if !state.Started {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionNotStarted,
				Message: "session not started",
			})
		}
		var payload EndPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: "PAYLOAD_DECODE_FAILED", Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		sessionID := strings.TrimSpace(payload.SessionID)
		if sessionID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionIDRequired,
				Message: "session id is required",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := EndPayload{SessionID: sessionID}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeEnded, "session", sessionID, payloadJSON, now().UTC())
		evt.SessionID = sessionID

		return command.Accept(evt)

	case commandTypeGateOpen:
		var payload GateOpenedPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: "PAYLOAD_DECODE_FAILED", Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		gateID := strings.TrimSpace(payload.GateID)
		gateType := strings.TrimSpace(payload.GateType)
		reason := strings.TrimSpace(payload.Reason)
		if gateID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionGateIDRequired,
				Message: "gate id is required",
			})
		}
		if gateType == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionGateTypeRequired,
				Message: "gate type is required",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := GateOpenedPayload{GateID: gateID, GateType: gateType, Reason: reason, Metadata: payload.Metadata}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeGateOpened, "session_gate", gateID, payloadJSON, now().UTC())

		return command.Accept(evt)

	case commandTypeGateResolve:
		var payload GateResolvedPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: "PAYLOAD_DECODE_FAILED", Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		gateID := strings.TrimSpace(payload.GateID)
		decision := strings.TrimSpace(payload.Decision)
		if gateID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionGateIDRequired,
				Message: "gate id is required",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := GateResolvedPayload{GateID: gateID, Decision: decision, Resolution: payload.Resolution}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeGateResolved, "session_gate", gateID, payloadJSON, now().UTC())

		return command.Accept(evt)

	case commandTypeGateAbandon:
		var payload GateAbandonedPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: "PAYLOAD_DECODE_FAILED", Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		gateID := strings.TrimSpace(payload.GateID)
		reason := strings.TrimSpace(payload.Reason)
		if gateID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionGateIDRequired,
				Message: "gate id is required",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := GateAbandonedPayload{GateID: gateID, Reason: reason}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeGateAbandoned, "session_gate", gateID, payloadJSON, now().UTC())

		return command.Accept(evt)

	case commandTypeSpotlightSet:
		var payload SpotlightSetPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: "PAYLOAD_DECODE_FAILED", Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		spotlightType := strings.TrimSpace(payload.SpotlightType)
		characterID := strings.TrimSpace(payload.CharacterID)
		if spotlightType == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionSpotlightTypeRequired,
				Message: "spotlight type is required",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := SpotlightSetPayload{SpotlightType: spotlightType, CharacterID: characterID}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeSpotlightSet, "session", cmd.SessionID, payloadJSON, now().UTC())

		return command.Accept(evt)

	case commandTypeSpotlightClear:
		var payload SpotlightClearedPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: "PAYLOAD_DECODE_FAILED", Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		reason := strings.TrimSpace(payload.Reason)
		if now == nil {
			now = time.Now
		}

		normalizedPayload := SpotlightClearedPayload{Reason: reason}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeSpotlightCleared, "session", cmd.SessionID, payloadJSON, now().UTC())

		return command.Accept(evt)

	default:
		return command.Reject(command.Rejection{Code: "COMMAND_TYPE_UNSUPPORTED", Message: fmt.Sprintf("command type %s is not supported by session decider", cmd.Type)})
	}
}
