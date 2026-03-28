package action

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	CommandTypeRollResolve   command.Type = "action.roll.resolve"
	CommandTypeOutcomeApply  command.Type = "action.outcome.apply"
	CommandTypeOutcomeReject command.Type = "action.outcome.reject"
	CommandTypeNoteAdd       command.Type = "story.note.add"

	EventTypeRollResolved    event.Type = "action.roll_resolved"
	EventTypeOutcomeApplied  event.Type = "action.outcome_applied"
	EventTypeOutcomeRejected event.Type = "action.outcome_rejected"
	EventTypeNoteAdded       event.Type = "story.note_added"

	rejectionCodeRequestIDRequired                 = "ACTION_REQUEST_ID_REQUIRED"
	rejectionCodeRollSeqRequired                   = "ACTION_ROLL_SEQ_REQUIRED"
	rejectionCodeOutcomeAlreadyApplied             = "ACTION_OUTCOME_ALREADY_APPLIED"
	rejectionCodeOutcomeEffectSystemOwnedForbidden = "ACTION_OUTCOME_EFFECT_SYSTEM_OWNED_FORBIDDEN"
	rejectionCodeOutcomeEffectTypeForbidden        = "ACTION_OUTCOME_EFFECT_TYPE_FORBIDDEN"
	rejectionCodeNoteContentRequired               = "ACTION_NOTE_CONTENT_REQUIRED"
)

var coreOutcomeEffectPolicy = newOutcomeEffectPolicy("session.gate_opened", "session.spotlight_set")

// RejectionCodes returns all rejection code strings used by the action
// decider. Used by startup validators to detect cross-domain collisions.
func RejectionCodes() []string {
	return []string{
		rejectionCodeRequestIDRequired,
		rejectionCodeRollSeqRequired,
		rejectionCodeOutcomeAlreadyApplied,
		rejectionCodeOutcomeEffectSystemOwnedForbidden,
		rejectionCodeOutcomeEffectTypeForbidden,
		rejectionCodeNoteContentRequired,
	}
}

// Decide returns the decision for an action command against current state.
//
// The action aggregate is intentionally lightweight: each supported action command
// becomes a typed domain event, keeping roll outcome logic and note-taking in one
// replayable stream.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.RequireNowFunc(now)
	switch cmd.Type {
	case CommandTypeRollResolve:
		return decideRollResolve(cmd, now)
	case CommandTypeOutcomeApply:
		return decideOutcomeApply(state, cmd, now)
	case CommandTypeOutcomeReject:
		return decideOutcomeReject(cmd, now)
	case CommandTypeNoteAdd:
		return decideNoteAdd(cmd, now)
	default:
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodeCommandTypeUnsupported,
			Message: "command type is not supported by action decider",
		})
	}
}
