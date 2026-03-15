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

	rejectionCodeRequestIDRequired                 = "REQUEST_ID_REQUIRED"
	rejectionCodeRollSeqRequired                   = "ROLL_SEQ_REQUIRED"
	rejectionCodeOutcomeAlreadyApplied             = "OUTCOME_ALREADY_APPLIED"
	rejectionCodeOutcomeEffectSystemOwnedForbidden = "OUTCOME_EFFECT_SYSTEM_OWNED_FORBIDDEN"
	rejectionCodeOutcomeEffectTypeForbidden        = "OUTCOME_EFFECT_TYPE_FORBIDDEN"
)

var coreOutcomeEffectPolicy = newOutcomeEffectPolicy("session.gate_opened", "session.spotlight_set")

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
			Code:    "COMMAND_TYPE_UNSUPPORTED",
			Message: "command type is not supported by action decider",
		})
	}
}
