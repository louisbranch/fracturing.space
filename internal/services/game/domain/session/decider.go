package session

import (
	"encoding/json"
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
	eventTypeStarted          event.Type   = "session.started"
	eventTypeEnded            event.Type   = "session.ended"
	eventTypeGateOpened       event.Type   = "session.gate_opened"
	eventTypeGateResolved     event.Type   = "session.gate_resolved"
	eventTypeGateAbandoned    event.Type   = "session.gate_abandoned"
	eventTypeSpotlightSet     event.Type   = "session.spotlight_set"
	eventTypeSpotlightCleared event.Type   = "session.spotlight_cleared"

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
	if cmd.Type == commandTypeStart {
		if state.Started {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionAlreadyStarted,
				Message: "session already started",
			})
		}
		var payload StartPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeStarted,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     sessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "session",
			EntityID:      sessionID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeEnd {
		if !state.Started {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeSessionNotStarted,
				Message: "session not started",
			})
		}
		var payload EndPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeEnded,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     sessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "session",
			EntityID:      sessionID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeGateOpen {
		var payload GateOpenedPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeGateOpened,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "session_gate",
			EntityID:      gateID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeGateResolve {
		var payload GateResolvedPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeGateResolved,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "session_gate",
			EntityID:      gateID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeGateAbandon {
		var payload GateAbandonedPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeGateAbandoned,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "session_gate",
			EntityID:      gateID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeSpotlightSet {
		var payload SpotlightSetPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeSpotlightSet,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "session",
			EntityID:      cmd.SessionID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeSpotlightClear {
		var payload SpotlightClearedPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		reason := strings.TrimSpace(payload.Reason)
		if now == nil {
			now = time.Now
		}

		normalizedPayload := SpotlightClearedPayload{Reason: reason}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeSpotlightCleared,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "session",
			EntityID:      cmd.SessionID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	return command.Decision{}
}
