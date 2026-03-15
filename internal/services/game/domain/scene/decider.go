package scene

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

const (
	CommandTypeCreate            command.Type = "scene.create"
	CommandTypeUpdate            command.Type = "scene.update"
	CommandTypeEnd               command.Type = "scene.end"
	CommandTypeCharacterAdd      command.Type = "scene.character.add"
	CommandTypeCharacterRemove   command.Type = "scene.character.remove"
	CommandTypeCharacterTransfer command.Type = "scene.character.transfer"
	CommandTypeTransition        command.Type = "scene.transition"
	CommandTypeGateOpen          command.Type = "scene.gate_open"
	CommandTypeGateResolve       command.Type = "scene.gate_resolve"
	CommandTypeGateAbandon       command.Type = "scene.gate_abandon"
	CommandTypeSpotlightSet      command.Type = "scene.spotlight_set"
	CommandTypeSpotlightClear    command.Type = "scene.spotlight_clear"

	EventTypeCreated          event.Type = "scene.created"
	EventTypeUpdated          event.Type = "scene.updated"
	EventTypeEnded            event.Type = "scene.ended"
	EventTypeCharacterAdded   event.Type = "scene.character_added"
	EventTypeCharacterRemoved event.Type = "scene.character_removed"
	EventTypeGateOpened       event.Type = "scene.gate_opened"
	EventTypeGateResolved     event.Type = "scene.gate_resolved"
	EventTypeGateAbandoned    event.Type = "scene.gate_abandoned"
	EventTypeSpotlightSet     event.Type = "scene.spotlight_set"
	EventTypeSpotlightCleared event.Type = "scene.spotlight_cleared"

	rejectionCodeSceneIDRequired         = "SCENE_ID_REQUIRED"
	rejectionCodeSceneNameRequired       = "SCENE_NAME_REQUIRED"
	rejectionCodeSceneCharactersRequired = "SCENE_CHARACTERS_REQUIRED"
	rejectionCodeSceneNotFound           = "SCENE_NOT_FOUND"
	rejectionCodeSceneNotActive          = "SCENE_NOT_ACTIVE"
	rejectionCodeSceneGateIDRequired     = "SCENE_GATE_ID_REQUIRED"
	rejectionCodeSceneGateTypeRequired   = "SCENE_GATE_TYPE_REQUIRED"
	rejectionCodeSceneGateAlreadyOpen    = "SCENE_GATE_ALREADY_OPEN"
	rejectionCodeSceneGateNotOpen        = "SCENE_GATE_NOT_OPEN"
	rejectionCodeCharacterIDRequired     = "SCENE_CHARACTER_ID_REQUIRED"
	rejectionCodeCharacterAlreadyInScene = "SCENE_CHARACTER_ALREADY_IN_SCENE"
	rejectionCodeCharacterNotInScene     = "SCENE_CHARACTER_NOT_IN_SCENE"
	rejectionCodeSpotlightTypeRequired   = "SCENE_SPOTLIGHT_TYPE_REQUIRED"
	rejectionCodeSpotlightNotSet         = "SCENE_SPOTLIGHT_NOT_SET"
	rejectionCodeSourceSceneIDRequired   = "SCENE_SOURCE_SCENE_ID_REQUIRED"
	rejectionCodeTargetSceneIDRequired   = "SCENE_TARGET_SCENE_ID_REQUIRED"
	rejectionCodeNewSceneIDRequired      = "SCENE_NEW_SCENE_ID_REQUIRED"
)

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
	default:
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodeCommandTypeUnsupported,
			Message: fmt.Sprintf("command type %s is not supported by scene decider", cmd.Type),
		})
	}
}
