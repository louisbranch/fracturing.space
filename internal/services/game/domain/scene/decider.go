// Package scene implements the scene domain aggregate for the game service.
//
// # Decider Signature Exception
//
// The scene Decide function takes map[ids.SceneID]State instead of a single
// State, unlike every other core domain decider. This is an intentional
// deviation required by cross-scene commands (CharacterTransfer, Transition)
// that must read and validate state from multiple scenes atomically within a
// single decision. The engine router (core_command_router.go) accounts for
// this by passing the full scenes map from aggregate state. Single-scene
// commands look up their target by ID from the map; cross-scene commands
// look up multiple entries.
package scene

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

const (
	CommandTypeCreate                      command.Type = "scene.create"
	CommandTypeUpdate                      command.Type = "scene.update"
	CommandTypeEnd                         command.Type = "scene.end"
	CommandTypeCharacterAdd                command.Type = "scene.character.add"
	CommandTypeCharacterRemove             command.Type = "scene.character.remove"
	CommandTypeCharacterTransfer           command.Type = "scene.character.transfer"
	CommandTypeTransition                  command.Type = "scene.transition"
	CommandTypeGateOpen                    command.Type = "scene.gate_open"
	CommandTypeGateResolve                 command.Type = "scene.gate_resolve"
	CommandTypeGateAbandon                 command.Type = "scene.gate_abandon"
	CommandTypeSpotlightSet                command.Type = "scene.spotlight_set"
	CommandTypeSpotlightClear              command.Type = "scene.spotlight_clear"
	CommandTypePlayerPhaseStart            command.Type = "scene.player_phase.start"
	CommandTypePlayerPhasePost             command.Type = "scene.player_phase.post"
	CommandTypePlayerPhaseYield            command.Type = "scene.player_phase.yield"
	CommandTypePlayerPhaseUnyield          command.Type = "scene.player_phase.unyield"
	CommandTypePlayerPhaseAccept           command.Type = "scene.player_phase.accept"
	CommandTypePlayerPhaseRequestRevisions command.Type = "scene.player_phase.request_revisions"
	CommandTypePlayerPhaseEnd              command.Type = "scene.player_phase.end"
	CommandTypeGMOutputCommit              command.Type = "scene.gm_output.commit"

	EventTypeCreated                       event.Type = "scene.created"
	EventTypeUpdated                       event.Type = "scene.updated"
	EventTypeEnded                         event.Type = "scene.ended"
	EventTypeCharacterAdded                event.Type = "scene.character_added"
	EventTypeCharacterRemoved              event.Type = "scene.character_removed"
	EventTypeGateOpened                    event.Type = "scene.gate_opened"
	EventTypeGateResolved                  event.Type = "scene.gate_resolved"
	EventTypeGateAbandoned                 event.Type = "scene.gate_abandoned"
	EventTypeSpotlightSet                  event.Type = "scene.spotlight_set"
	EventTypeSpotlightCleared              event.Type = "scene.spotlight_cleared"
	EventTypePlayerPhaseStarted            event.Type = "scene.player_phase_started"
	EventTypePlayerPhasePosted             event.Type = "scene.player_phase_posted"
	EventTypePlayerPhaseYielded            event.Type = "scene.player_phase_yielded"
	EventTypePlayerPhaseReviewStarted      event.Type = "scene.player_phase_review_started"
	EventTypePlayerPhaseUnyielded          event.Type = "scene.player_phase_unyielded"
	EventTypePlayerPhaseRevisionsRequested event.Type = "scene.player_phase_revisions_requested"
	EventTypePlayerPhaseAccepted           event.Type = "scene.player_phase_accepted"
	EventTypePlayerPhaseEnded              event.Type = "scene.player_phase_ended"
	EventTypeGMOutputCommitted             event.Type = "scene.gm_output_committed"

	rejectionCodeSceneIDRequired                      = "SCENE_ID_REQUIRED"
	rejectionCodeSceneNameRequired                    = "SCENE_NAME_REQUIRED"
	rejectionCodeSceneCharactersRequired              = "SCENE_CHARACTERS_REQUIRED"
	rejectionCodeSceneNotFound                        = "SCENE_NOT_FOUND"
	rejectionCodeSceneNotActive                       = "SCENE_NOT_ACTIVE"
	rejectionCodeSceneGateIDRequired                  = "SCENE_GATE_ID_REQUIRED"
	rejectionCodeSceneGateTypeRequired                = "SCENE_GATE_TYPE_REQUIRED"
	rejectionCodeSceneGateAlreadyOpen                 = "SCENE_GATE_ALREADY_OPEN"
	rejectionCodeSceneGateNotOpen                     = "SCENE_GATE_NOT_OPEN"
	rejectionCodeCharacterIDRequired                  = "SCENE_CHARACTER_ID_REQUIRED"
	rejectionCodeCharacterAlreadyInScene              = "SCENE_CHARACTER_ALREADY_IN_SCENE"
	rejectionCodeCharacterNotInScene                  = "SCENE_CHARACTER_NOT_IN_SCENE"
	rejectionCodeSpotlightTypeRequired                = "SCENE_SPOTLIGHT_TYPE_REQUIRED"
	rejectionCodeSpotlightNotSet                      = "SCENE_SPOTLIGHT_NOT_SET"
	rejectionCodeSourceSceneIDRequired                = "SCENE_SOURCE_SCENE_ID_REQUIRED"
	rejectionCodeTargetSceneIDRequired                = "SCENE_TARGET_SCENE_ID_REQUIRED"
	rejectionCodeNewSceneIDRequired                   = "SCENE_NEW_SCENE_ID_REQUIRED"
	rejectionCodeScenePlayerPhaseIDRequired           = "SCENE_PLAYER_PHASE_ID_REQUIRED"
	rejectionCodeScenePlayerPhaseAlreadyOpen          = "SCENE_PLAYER_PHASE_ALREADY_OPEN"
	rejectionCodeScenePlayerPhaseNotOpen              = "SCENE_PLAYER_PHASE_NOT_OPEN"
	rejectionCodeScenePlayerPhaseNotInPlayersState    = "SCENE_PLAYER_PHASE_NOT_IN_PLAYERS_STATE"
	rejectionCodeScenePlayerPhaseNotInReviewState     = "SCENE_PLAYER_PHASE_NOT_IN_REVIEW_STATE"
	rejectionCodeScenePlayerPhaseParticipantRequired  = "SCENE_PLAYER_PHASE_PARTICIPANT_REQUIRED"
	rejectionCodeScenePlayerPhaseParticipantNotActing = "SCENE_PLAYER_PHASE_PARTICIPANT_NOT_ACTING"
	rejectionCodeScenePlayerPhaseSummaryRequired      = "SCENE_PLAYER_PHASE_SUMMARY_REQUIRED"
	rejectionCodeScenePlayerPhaseRevisionRequired     = "SCENE_PLAYER_PHASE_REVISION_REQUIRED"
	rejectionCodeSceneGMOutputRequired                = "SCENE_GM_OUTPUT_REQUIRED"
	rejectionCodeSceneGMOutputParticipantRequired     = "SCENE_GM_OUTPUT_PARTICIPANT_REQUIRED"
)

// RejectionCodes returns all rejection code strings used by the scene
// decider. Used by startup validators to detect cross-domain collisions.
func RejectionCodes() []string {
	return []string{
		rejectionCodeSceneIDRequired,
		rejectionCodeSceneNameRequired,
		rejectionCodeSceneCharactersRequired,
		rejectionCodeSceneNotFound,
		rejectionCodeSceneNotActive,
		rejectionCodeSceneGateIDRequired,
		rejectionCodeSceneGateTypeRequired,
		rejectionCodeSceneGateAlreadyOpen,
		rejectionCodeSceneGateNotOpen,
		rejectionCodeCharacterIDRequired,
		rejectionCodeCharacterAlreadyInScene,
		rejectionCodeCharacterNotInScene,
		rejectionCodeSpotlightTypeRequired,
		rejectionCodeSpotlightNotSet,
		rejectionCodeSourceSceneIDRequired,
		rejectionCodeTargetSceneIDRequired,
		rejectionCodeNewSceneIDRequired,
		rejectionCodeScenePlayerPhaseIDRequired,
		rejectionCodeScenePlayerPhaseAlreadyOpen,
		rejectionCodeScenePlayerPhaseNotOpen,
		rejectionCodeScenePlayerPhaseNotInPlayersState,
		rejectionCodeScenePlayerPhaseNotInReviewState,
		rejectionCodeScenePlayerPhaseParticipantRequired,
		rejectionCodeScenePlayerPhaseParticipantNotActing,
		rejectionCodeScenePlayerPhaseSummaryRequired,
		rejectionCodeScenePlayerPhaseRevisionRequired,
		rejectionCodeSceneGMOutputRequired,
		rejectionCodeSceneGMOutputParticipantRequired,
	}
}

// Decide returns the decision for a scene command against the current scene
// states. The scenes map contains all scene states keyed by scene ID.
//
// Commands that operate on a single scene look up the target scene by ID.
// Cross-scene commands (transfer, transition) look up multiple scenes.
func Decide(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)
	switch cmd.Type {
	case CommandTypeCreate:
		return decideCreate(cmd, now)
	case CommandTypeUpdate:
		return decideUpdate(scenes, cmd, now)
	case CommandTypeEnd:
		return decideEnd(scenes, cmd, now)
	case CommandTypeCharacterAdd:
		return decideCharacterAdd(scenes, cmd, now)
	case CommandTypeCharacterRemove:
		return decideCharacterRemove(scenes, cmd, now)
	case CommandTypeCharacterTransfer:
		return decideCharacterTransfer(scenes, cmd, now)
	case CommandTypeTransition:
		return decideTransition(scenes, cmd, now)
	case CommandTypeGateOpen:
		return decideGateOpen(scenes, cmd, now)
	case CommandTypeGateResolve:
		return decideGateResolve(scenes, cmd, now)
	case CommandTypeGateAbandon:
		return decideGateAbandon(scenes, cmd, now)
	case CommandTypeSpotlightSet:
		return decideSpotlightSet(scenes, cmd, now)
	case CommandTypeSpotlightClear:
		return decideSpotlightClear(scenes, cmd, now)
	case CommandTypePlayerPhaseStart:
		return decidePlayerPhaseStart(scenes, cmd, now)
	case CommandTypePlayerPhasePost:
		return decidePlayerPhasePost(scenes, cmd, now)
	case CommandTypePlayerPhaseYield:
		return decidePlayerPhaseYield(scenes, cmd, now)
	case CommandTypePlayerPhaseUnyield:
		return decidePlayerPhaseUnyield(scenes, cmd, now)
	case CommandTypePlayerPhaseAccept:
		return decidePlayerPhaseAccept(scenes, cmd, now)
	case CommandTypePlayerPhaseRequestRevisions:
		return decidePlayerPhaseRequestRevisions(scenes, cmd, now)
	case CommandTypePlayerPhaseEnd:
		return decidePlayerPhaseEnd(scenes, cmd, now)
	case CommandTypeGMOutputCommit:
		return decideGMOutputCommit(scenes, cmd, now)
	default:
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodeCommandTypeUnsupported,
			Message: fmt.Sprintf("command type %s is not supported by scene decider", cmd.Type),
		})
	}
}
