package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

const (
	CommandTypeStart              command.Type = "session.start"
	CommandTypeEnd                command.Type = "session.end"
	CommandTypeGateOpen           command.Type = "session.gate_open"
	CommandTypeGateRespond        command.Type = "session.gate_record_response"
	CommandTypeGateResolve        command.Type = "session.gate_resolve"
	CommandTypeGateAbandon        command.Type = "session.gate_abandon"
	CommandTypeSpotlightSet       command.Type = "session.spotlight_set"
	CommandTypeSpotlightClear     command.Type = "session.spotlight_clear"
	EventTypeStarted              event.Type   = "session.started"
	EventTypeEnded                event.Type   = "session.ended"
	EventTypeGateOpened           event.Type   = "session.gate_opened"
	EventTypeGateResponseRecorded event.Type   = "session.gate_response_recorded"
	EventTypeGateResolved         event.Type   = "session.gate_resolved"
	EventTypeGateAbandoned        event.Type   = "session.gate_abandoned"
	EventTypeSpotlightSet         event.Type   = "session.spotlight_set"
	EventTypeSpotlightCleared     event.Type   = "session.spotlight_cleared"

	rejectionCodeSessionIDRequired              = "SESSION_ID_REQUIRED"
	rejectionCodeSessionAlreadyStarted          = "SESSION_ALREADY_STARTED"
	rejectionCodeSessionNotStarted              = "SESSION_NOT_STARTED"
	rejectionCodeSessionGateIDRequired          = "SESSION_GATE_ID_REQUIRED"
	rejectionCodeSessionGateTypeRequired        = "SESSION_GATE_TYPE_REQUIRED"
	rejectionCodeSessionGateParticipantRequired = "SESSION_GATE_PARTICIPANT_REQUIRED"
	rejectionCodeSessionSpotlightTypeRequired   = "SESSION_SPOTLIGHT_TYPE_REQUIRED"
)

// Decide returns the decision for a session command against current state.
//
// It maps every supported session lifecycle and gate command to deterministic
// events, and leaves status checks to replayable state transitions rather than
// imperative side effects.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)

	switch cmd.Type {
	case CommandTypeStart:
		if state.Started {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionAlreadyStarted,
				Message: "session already started",
			})
		}
		var payload StartPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		sessionID := strings.TrimSpace(payload.SessionID.String())
		if sessionID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionIDRequired,
				Message: "session id is required",
			})
		}
		sessionName := strings.TrimSpace(payload.SessionName)

		normalizedPayload := StartPayload{SessionID: ids.SessionID(sessionID), SessionName: sessionName}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeStarted, "session", sessionID, payloadJSON, now().UTC())
		evt.SessionID = ids.SessionID(sessionID)

		return command.Accept(evt)

	case CommandTypeEnd:
		if !state.Started {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionNotStarted,
				Message: "session not started",
			})
		}
		var payload EndPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		sessionID := strings.TrimSpace(payload.SessionID.String())
		if sessionID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionIDRequired,
				Message: "session id is required",
			})
		}

		normalizedPayload := EndPayload{SessionID: ids.SessionID(sessionID)}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeEnded, "session", sessionID, payloadJSON, now().UTC())
		evt.SessionID = ids.SessionID(sessionID)

		return command.Accept(evt)

	case CommandTypeGateOpen:
		var payload GateOpenedPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		gateID := strings.TrimSpace(payload.GateID.String())
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

		normalizedPayload := GateOpenedPayload{GateID: ids.GateID(gateID), GateType: gateType, Reason: reason, Metadata: payload.Metadata}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeGateOpened, "session_gate", gateID, payloadJSON, now().UTC())

		return command.Accept(evt)

	case CommandTypeGateResolve:
		return module.DecideFunc(
			cmd,
			EventTypeGateResolved,
			"session_gate",
			func(payload *GateResolvedPayload) string {
				return payload.GateID.String()
			},
			func(payload *GateResolvedPayload, _ func() time.Time) *command.Rejection {
				payload.GateID = ids.GateID(strings.TrimSpace(payload.GateID.String()))
				payload.Decision = strings.TrimSpace(payload.Decision)
				if payload.GateID == "" {
					return &command.Rejection{
						Code:    rejectionCodeSessionGateIDRequired,
						Message: "gate id is required",
					}
				}
				return nil
			},
			now,
		)

	case CommandTypeGateRespond:
		return module.DecideFunc(
			cmd,
			EventTypeGateResponseRecorded,
			"session_gate",
			func(payload *GateResponseRecordedPayload) string {
				return payload.GateID.String()
			},
			func(payload *GateResponseRecordedPayload, _ func() time.Time) *command.Rejection {
				payload.GateID = ids.GateID(strings.TrimSpace(payload.GateID.String()))
				payload.ParticipantID = ids.ParticipantID(strings.TrimSpace(payload.ParticipantID.String()))
				payload.Decision = strings.TrimSpace(payload.Decision)
				if payload.GateID == "" {
					return &command.Rejection{
						Code:    rejectionCodeSessionGateIDRequired,
						Message: "gate id is required",
					}
				}
				if payload.ParticipantID == "" {
					return &command.Rejection{
						Code:    rejectionCodeSessionGateParticipantRequired,
						Message: "participant id is required",
					}
				}
				return nil
			},
			now,
		)

	case CommandTypeGateAbandon:
		return module.DecideFunc(
			cmd,
			EventTypeGateAbandoned,
			"session_gate",
			func(payload *GateAbandonedPayload) string {
				return payload.GateID.String()
			},
			func(payload *GateAbandonedPayload, _ func() time.Time) *command.Rejection {
				payload.GateID = ids.GateID(strings.TrimSpace(payload.GateID.String()))
				payload.Reason = strings.TrimSpace(payload.Reason)
				if payload.GateID == "" {
					return &command.Rejection{
						Code:    rejectionCodeSessionGateIDRequired,
						Message: "gate id is required",
					}
				}
				return nil
			},
			now,
		)

	case CommandTypeSpotlightSet:
		var payload SpotlightSetPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
		}
		spotlightType := strings.TrimSpace(payload.SpotlightType)
		characterID := strings.TrimSpace(payload.CharacterID.String())
		if spotlightType == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionSpotlightTypeRequired,
				Message: "spotlight type is required",
			})
		}

		normalizedPayload := SpotlightSetPayload{SpotlightType: spotlightType, CharacterID: ids.CharacterID(characterID)}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := command.NewEvent(cmd, EventTypeSpotlightSet, "session", cmd.SessionID.String(), payloadJSON, now().UTC())

		return command.Accept(evt)

	case CommandTypeSpotlightClear:
		return module.DecideFunc(
			cmd,
			EventTypeSpotlightCleared,
			"session",
			func(_ *SpotlightClearedPayload) string {
				return cmd.SessionID.String()
			},
			func(payload *SpotlightClearedPayload, _ func() time.Time) *command.Rejection {
				payload.Reason = strings.TrimSpace(payload.Reason)
				return nil
			},
			now,
		)

	default:
		return command.Reject(command.Rejection{Code: command.RejectionCodeCommandTypeUnsupported, Message: fmt.Sprintf("command type %s is not supported by session decider", cmd.Type)})
	}
}
