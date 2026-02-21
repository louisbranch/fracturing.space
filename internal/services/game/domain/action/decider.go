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

	rejectionCodeRequestIDRequired     = "REQUEST_ID_REQUIRED"
	rejectionCodeRollSeqRequired       = "ROLL_SEQ_REQUIRED"
	rejectionCodeOutcomeAlreadyApplied = "OUTCOME_ALREADY_APPLIED"
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
		if _, alreadyApplied := state.AppliedOutcomes[payload.RollSeq]; alreadyApplied {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeOutcomeAlreadyApplied,
				Message: ErrOutcomeAlreadyApplied.Error(),
			})
		}
		return acceptActionEvent(cmd, now, eventTypeOutcomeApplied, "outcome", requestID, payload)
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
