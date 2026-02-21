package action

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeRollResolve   command.Type = "action.roll.resolve"
	commandTypeOutcomeApply  command.Type = "action.outcome.apply"
	commandTypeOutcomeReject command.Type = "action.outcome.reject"
	commandTypeNoteAdd       command.Type = "story.note.add"

	eventTypeRollResolved    event.Type = "action.roll_resolved"
	eventTypeOutcomeApplied  event.Type = "action.outcome_applied"
	eventTypeOutcomeRejected event.Type = "action.outcome_rejected"
	eventTypeNoteAdded       event.Type = "story.note_added"

	rejectionCodeRequestIDRequired                 = "REQUEST_ID_REQUIRED"
	rejectionCodeRollSeqRequired                   = "ROLL_SEQ_REQUIRED"
	rejectionCodeOutcomeAlreadyApplied             = "OUTCOME_ALREADY_APPLIED"
	rejectionCodeOutcomeEffectSystemOwnedForbidden = "OUTCOME_EFFECT_SYSTEM_OWNED_FORBIDDEN"
)

// Decide returns the decision for an action command against current state.
//
// The action aggregate is intentionally lightweight: each supported action command
// becomes a typed domain event, keeping roll outcome logic and note-taking in one
// replayable stream.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	if now == nil {
		now = time.Now
	}

	switch cmd.Type {
	case commandTypeRollResolve:
		var payload RollResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		requestID := strings.TrimSpace(payload.RequestID)
		if requestID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeRequestIDRequired,
				Message: "request_id is required",
			})
		}
		if payload.RollSeq == 0 {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeRollSeqRequired,
				Message: "roll_seq must be greater than zero",
			})
		}
		return acceptActionEvent(cmd, now, eventTypeRollResolved, "roll", requestID, payload)
	case commandTypeOutcomeApply:
		var payload OutcomeApplyPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		requestID := strings.TrimSpace(payload.RequestID)
		if requestID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeRequestIDRequired,
				Message: "request_id is required",
			})
		}
		if payload.RollSeq == 0 {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeRollSeqRequired,
				Message: "roll_seq must be greater than zero",
			})
		}
		if hasSystemOwnedOutcomeEffect(payload.PreEffects) || hasSystemOwnedOutcomeEffect(payload.PostEffects) {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeOutcomeEffectSystemOwnedForbidden,
				Message: "core action.outcome.apply cannot emit system-owned effects",
			})
		}
		if _, alreadyApplied := state.AppliedOutcomes[payload.RollSeq]; alreadyApplied {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeOutcomeAlreadyApplied,
				Message: ErrOutcomeAlreadyApplied.Error(),
			})
		}
		events := make([]event.Event, 0, len(payload.PreEffects)+len(payload.PostEffects)+1)
		for _, effect := range payload.PreEffects {
			events = append(events, buildOutcomeEffectEvent(cmd, now, effect))
		}

		postEffects := payload.PostEffects
		payload.PreEffects = nil
		payload.PostEffects = nil
		outcomeEvent := acceptActionEvent(cmd, now, eventTypeOutcomeApplied, "outcome", requestID, payload).Events
		events = append(events, outcomeEvent...)

		for _, effect := range postEffects {
			events = append(events, buildOutcomeEffectEvent(cmd, now, effect))
		}
		return command.Accept(events...)
	case commandTypeOutcomeReject:
		var payload OutcomeRejectPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		requestID := strings.TrimSpace(payload.RequestID)
		if requestID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeRequestIDRequired,
				Message: "request_id is required",
			})
		}
		if payload.RollSeq == 0 {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeRollSeqRequired,
				Message: "roll_seq must be greater than zero",
			})
		}
		return acceptActionEvent(cmd, now, eventTypeOutcomeRejected, "outcome", requestID, payload)
	case commandTypeNoteAdd:
		var payload NoteAddPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		return acceptActionEvent(cmd, now, eventTypeNoteAdded, "note", cmd.EntityID, payload)
	default:
		return command.Reject(command.Rejection{
			Code:    "COMMAND_TYPE_UNSUPPORTED",
			Message: "command type is not supported by action decider",
		})
	}
}

// acceptActionEvent creates the standard action event envelope for accepted commands.
//
// Centralizing this constructor keeps action event metadata consistent even when
// specific systems add new action shapes.
func acceptActionEvent(cmd command.Command, now func() time.Time, eventType event.Type, entityType, entityID string, payload any) command.Decision {
	payloadJSON, _ := json.Marshal(payload)
	evt := event.Event{
		CampaignID:    cmd.CampaignID,
		Type:          eventType,
		Timestamp:     now().UTC(),
		ActorType:     event.ActorType(cmd.ActorType),
		ActorID:       cmd.ActorID,
		SessionID:     cmd.SessionID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		EntityType:    entityType,
		EntityID:      entityID,
		SystemID:      cmd.SystemID,
		SystemVersion: cmd.SystemVersion,
		CorrelationID: cmd.CorrelationID,
		CausationID:   cmd.CausationID,
		PayloadJSON:   payloadJSON,
	}

	return command.Accept(evt)
}

func buildOutcomeEffectEvent(cmd command.Command, now func() time.Time, effect OutcomeAppliedEffect) event.Event {
	payloadJSON := effect.PayloadJSON
	if len(payloadJSON) == 0 {
		payloadJSON = []byte("{}")
	}
	return event.Event{
		CampaignID:    cmd.CampaignID,
		Type:          event.Type(strings.TrimSpace(effect.Type)),
		Timestamp:     now().UTC(),
		ActorType:     event.ActorType(cmd.ActorType),
		ActorID:       cmd.ActorID,
		SessionID:     cmd.SessionID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		EntityType:    strings.TrimSpace(effect.EntityType),
		EntityID:      strings.TrimSpace(effect.EntityID),
		SystemID:      strings.TrimSpace(effect.SystemID),
		SystemVersion: strings.TrimSpace(effect.SystemVersion),
		CorrelationID: cmd.CorrelationID,
		CausationID:   cmd.CausationID,
		PayloadJSON:   payloadJSON,
	}
}

func hasSystemOwnedOutcomeEffect(effects []OutcomeAppliedEffect) bool {
	for _, effect := range effects {
		if strings.HasPrefix(strings.TrimSpace(effect.Type), "sys.") {
			return true
		}
		if strings.TrimSpace(effect.SystemID) != "" || strings.TrimSpace(effect.SystemVersion) != "" {
			return true
		}
	}
	return false
}
