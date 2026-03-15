package session

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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
	rejectionCodeSessionGateAlreadyOpen         = "SESSION_GATE_ALREADY_OPEN"
	rejectionCodeSessionGateMetadataInvalid     = "SESSION_GATE_METADATA_INVALID"
	rejectionCodeSessionGateNotOpen             = "SESSION_GATE_NOT_OPEN"
	rejectionCodeSessionGateMismatch            = "SESSION_GATE_MISMATCH"
	rejectionCodeSessionGateResponseInvalid     = "SESSION_GATE_RESPONSE_INVALID"
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
		return decideStart(state, cmd, now)

	case CommandTypeEnd:
		return decideEnd(state, cmd, now)

	case CommandTypeGateOpen:
		return decideGateOpen(state, cmd, now)

	case CommandTypeGateResolve:
		return decideGateResolve(state, cmd, now)

	case CommandTypeGateRespond:
		return decideGateRespond(state, cmd, now)

	case CommandTypeGateAbandon:
		return decideGateAbandon(state, cmd, now)

	case CommandTypeSpotlightSet:
		return decideSpotlightSet(cmd, now)

	case CommandTypeSpotlightClear:
		return decideSpotlightClear(cmd, now)

	default:
		return command.Reject(command.Rejection{Code: command.RejectionCodeCommandTypeUnsupported, Message: fmt.Sprintf("command type %s is not supported by session decider", cmd.Type)})
	}
}
