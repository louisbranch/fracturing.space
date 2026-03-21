package session

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	CommandTypeStart                  command.Type = "session.start"
	CommandTypeEnd                    command.Type = "session.end"
	CommandTypeGateOpen               command.Type = "session.gate_open"
	CommandTypeGateRespond            command.Type = "session.gate_record_response"
	CommandTypeGateResolve            command.Type = "session.gate_resolve"
	CommandTypeGateAbandon            command.Type = "session.gate_abandon"
	CommandTypeSpotlightSet           command.Type = "session.spotlight_set"
	CommandTypeSpotlightClear         command.Type = "session.spotlight_clear"
	CommandTypeActiveSceneSet         command.Type = "session.active_scene.set"
	CommandTypeGMAuthoritySet         command.Type = "session.gm_authority.set"
	CommandTypeOOCPause               command.Type = "session.ooc.pause"
	CommandTypeOOCPost                command.Type = "session.ooc.post"
	CommandTypeOOCReadyMark           command.Type = "session.ooc.ready_mark"
	CommandTypeOOCReadyClear          command.Type = "session.ooc.ready_clear"
	CommandTypeOOCResume              command.Type = "session.ooc.resume"
	CommandTypeOOCInterruptionResolve command.Type = "session.ooc.interruption_resolve"
	CommandTypeAITurnQueue            command.Type = "session.ai_turn.queue"
	CommandTypeAITurnStart            command.Type = "session.ai_turn.start"
	CommandTypeAITurnFail             command.Type = "session.ai_turn.fail"
	CommandTypeAITurnClear            command.Type = "session.ai_turn.clear"
	EventTypeStarted                  event.Type   = "session.started"
	EventTypeEnded                    event.Type   = "session.ended"
	EventTypeGateOpened               event.Type   = "session.gate_opened"
	EventTypeGateResponseRecorded     event.Type   = "session.gate_response_recorded"
	EventTypeGateResolved             event.Type   = "session.gate_resolved"
	EventTypeGateAbandoned            event.Type   = "session.gate_abandoned"
	EventTypeSpotlightSet             event.Type   = "session.spotlight_set"
	EventTypeSpotlightCleared         event.Type   = "session.spotlight_cleared"
	EventTypeActiveSceneSet           event.Type   = "session.active_scene_set"
	EventTypeGMAuthoritySet           event.Type   = "session.gm_authority_set"
	EventTypeOOCPaused                event.Type   = "session.ooc_paused"
	EventTypeOOCPosted                event.Type   = "session.ooc_posted"
	EventTypeOOCReadyMarked           event.Type   = "session.ooc_ready_marked"
	EventTypeOOCReadyCleared          event.Type   = "session.ooc_ready_cleared"
	EventTypeOOCResumed               event.Type   = "session.ooc_resumed"
	EventTypeOOCInterruptionResolved  event.Type   = "session.ooc_interruption_resolved"
	EventTypeAITurnQueued             event.Type   = "session.ai_turn_queued"
	EventTypeAITurnRunning            event.Type   = "session.ai_turn_running"
	EventTypeAITurnFailed             event.Type   = "session.ai_turn_failed"
	EventTypeAITurnCleared            event.Type   = "session.ai_turn_cleared"

	rejectionCodeSessionIDRequired              = "SESSION_ID_REQUIRED"
	rejectionCodeSessionNameRequired            = "SESSION_NAME_REQUIRED"
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
	rejectionCodeSessionSpotlightTypeInvalid    = "SESSION_SPOTLIGHT_TYPE_INVALID"
	rejectionCodeSessionSpotlightTargetInvalid  = "SESSION_SPOTLIGHT_TARGET_INVALID"
	rejectionCodeSessionActiveSceneRequired     = "SESSION_ACTIVE_SCENE_REQUIRED"
	rejectionCodeSessionActiveSceneUnchanged    = "SESSION_ACTIVE_SCENE_UNCHANGED"
	rejectionCodeSessionGMAuthorityRequired     = "SESSION_GM_AUTHORITY_REQUIRED"
	rejectionCodeSessionGMAuthorityUnchanged    = "SESSION_GM_AUTHORITY_UNCHANGED"
	rejectionCodeSessionOOCAlreadyOpen          = "SESSION_OOC_ALREADY_OPEN"
	rejectionCodeSessionOOCNotOpen              = "SESSION_OOC_NOT_OPEN"
	rejectionCodeSessionOOCResolutionNotPending = "SESSION_OOC_RESOLUTION_NOT_PENDING"
	rejectionCodeSessionOOCPostIDRequired       = "SESSION_OOC_POST_ID_REQUIRED"
	rejectionCodeSessionOOCParticipantRequired  = "SESSION_OOC_PARTICIPANT_REQUIRED"
	rejectionCodeSessionOOCBodyRequired         = "SESSION_OOC_BODY_REQUIRED"
	rejectionCodeSessionAITurnTokenRequired     = "SESSION_AI_TURN_TOKEN_REQUIRED"
	rejectionCodeSessionAITurnOwnerRequired     = "SESSION_AI_TURN_OWNER_REQUIRED"
	rejectionCodeSessionAITurnNotQueued         = "SESSION_AI_TURN_NOT_QUEUED"
	rejectionCodeSessionAITurnNotActive         = "SESSION_AI_TURN_NOT_ACTIVE"
	rejectionCodeSessionAITurnTokenMismatch     = "SESSION_AI_TURN_TOKEN_MISMATCH"
)

// RejectionCodes returns all rejection code strings used by the session
// decider. Used by startup validators to detect cross-domain collisions.
func RejectionCodes() []string {
	return []string{
		rejectionCodeSessionIDRequired,
		rejectionCodeSessionNameRequired,
		rejectionCodeSessionAlreadyStarted,
		rejectionCodeSessionNotStarted,
		rejectionCodeSessionGateIDRequired,
		rejectionCodeSessionGateTypeRequired,
		rejectionCodeSessionGateParticipantRequired,
		rejectionCodeSessionGateAlreadyOpen,
		rejectionCodeSessionGateMetadataInvalid,
		rejectionCodeSessionGateNotOpen,
		rejectionCodeSessionGateMismatch,
		rejectionCodeSessionGateResponseInvalid,
		rejectionCodeSessionSpotlightTypeInvalid,
		rejectionCodeSessionSpotlightTargetInvalid,
		rejectionCodeSessionActiveSceneRequired,
		rejectionCodeSessionActiveSceneUnchanged,
		rejectionCodeSessionGMAuthorityRequired,
		rejectionCodeSessionGMAuthorityUnchanged,
		rejectionCodeSessionOOCAlreadyOpen,
		rejectionCodeSessionOOCNotOpen,
		rejectionCodeSessionOOCResolutionNotPending,
		rejectionCodeSessionOOCPostIDRequired,
		rejectionCodeSessionOOCParticipantRequired,
		rejectionCodeSessionOOCBodyRequired,
		rejectionCodeSessionAITurnTokenRequired,
		rejectionCodeSessionAITurnOwnerRequired,
		rejectionCodeSessionAITurnNotQueued,
		rejectionCodeSessionAITurnNotActive,
		rejectionCodeSessionAITurnTokenMismatch,
	}
}

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

	case CommandTypeActiveSceneSet:
		return decideActiveSceneSet(state, cmd, now)

	case CommandTypeGMAuthoritySet:
		return decideGMAuthoritySet(state, cmd, now)

	case CommandTypeOOCPause:
		return decideOOCPause(state, cmd, now)

	case CommandTypeOOCPost:
		return decideOOCPost(state, cmd, now)

	case CommandTypeOOCReadyMark:
		return decideOOCReadyMark(state, cmd, now)

	case CommandTypeOOCReadyClear:
		return decideOOCReadyClear(state, cmd, now)

	case CommandTypeOOCResume:
		return decideOOCResume(state, cmd, now)

	case CommandTypeOOCInterruptionResolve:
		return decideOOCInterruptionResolve(state, cmd, now)

	case CommandTypeAITurnQueue:
		return decideAITurnQueue(state, cmd, now)

	case CommandTypeAITurnStart:
		return decideAITurnStart(state, cmd, now)

	case CommandTypeAITurnFail:
		return decideAITurnFail(state, cmd, now)

	case CommandTypeAITurnClear:
		return decideAITurnClear(state, cmd, now)

	default:
		return command.Reject(command.Rejection{Code: command.RejectionCodeCommandTypeUnsupported, Message: fmt.Sprintf("command type %s is not supported by session decider", cmd.Type)})
	}
}
