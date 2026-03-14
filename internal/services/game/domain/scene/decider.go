package scene

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
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

func decideCreate(cmd command.Command, now func() time.Time) command.Decision {
	var payload CreatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneNameRequired, Message: "scene name is required"})
	}
	description := strings.TrimSpace(payload.Description)

	charIDs := normalizeCharacterIDs(payload.CharacterIDs)
	if len(charIDs) == 0 {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneCharactersRequired, Message: "at least one character is required"})
	}

	ts := now().UTC()
	events := make([]event.Event, 0, 1+len(charIDs))

	normalizedCreate := CreatePayload{
		SceneID:      ids.SceneID(sceneID),
		Name:         name,
		Description:  description,
		CharacterIDs: charIDs,
	}
	createJSON, _ := json.Marshal(normalizedCreate)
	events = append(events, command.NewEvent(cmd, EventTypeCreated, "scene", sceneID, createJSON, ts))

	for _, charID := range charIDs {
		addPayload := CharacterAddedPayload{SceneID: ids.SceneID(sceneID), CharacterID: charID}
		addJSON, _ := json.Marshal(addPayload)
		events = append(events, command.NewEvent(cmd, EventTypeCharacterAdded, "scene", sceneID, addJSON, ts))
	}

	return command.Accept(events...)
}

func decideUpdate(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload UpdatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}

	normalized := UpdatePayload{
		SceneID:     ids.SceneID(sceneID),
		Name:        strings.TrimSpace(payload.Name),
		Description: strings.TrimSpace(payload.Description),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeUpdated, "scene", sceneID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideEnd(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload EndPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}

	normalized := EndPayload{SceneID: ids.SceneID(sceneID), Reason: strings.TrimSpace(payload.Reason)}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeEnded, "scene", sceneID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideCharacterAdd(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload CharacterAddedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeCharacterIDRequired, Message: "character id is required"})
	}
	scene := scenes[ids.SceneID(sceneID)]
	if scene.Characters[ids.CharacterID(characterID)] {
		return command.Reject(command.Rejection{Code: rejectionCodeCharacterAlreadyInScene, Message: "character is already in scene"})
	}

	normalized := CharacterAddedPayload{SceneID: ids.SceneID(sceneID), CharacterID: ids.CharacterID(characterID)}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeCharacterAdded, "scene", sceneID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideCharacterRemove(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload CharacterRemovedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeCharacterIDRequired, Message: "character id is required"})
	}
	scene := scenes[ids.SceneID(sceneID)]
	if !scene.Characters[ids.CharacterID(characterID)] {
		return command.Reject(command.Rejection{Code: rejectionCodeCharacterNotInScene, Message: "character is not in scene"})
	}

	normalized := CharacterRemovedPayload{SceneID: ids.SceneID(sceneID), CharacterID: ids.CharacterID(characterID)}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeCharacterRemoved, "scene", sceneID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideCharacterTransfer(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload CharacterTransferPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sourceSceneID := strings.TrimSpace(payload.SourceSceneID.String())
	if sourceSceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSourceSceneIDRequired, Message: "source scene id is required"})
	}
	targetSceneID := strings.TrimSpace(payload.TargetSceneID.String())
	if targetSceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeTargetSceneIDRequired, Message: "target scene id is required"})
	}
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeCharacterIDRequired, Message: "character id is required"})
	}
	if rejection := requireActiveScene(scenes, sourceSceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	if rejection := requireActiveScene(scenes, targetSceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	source := scenes[ids.SceneID(sourceSceneID)]
	if !source.Characters[ids.CharacterID(characterID)] {
		return command.Reject(command.Rejection{Code: rejectionCodeCharacterNotInScene, Message: "character is not in source scene"})
	}

	ts := now().UTC()
	removePayload := CharacterRemovedPayload{SceneID: ids.SceneID(sourceSceneID), CharacterID: ids.CharacterID(characterID)}
	removeJSON, _ := json.Marshal(removePayload)
	removeEvt := command.NewEvent(cmd, EventTypeCharacterRemoved, "scene", sourceSceneID, removeJSON, ts)

	addPayload := CharacterAddedPayload{SceneID: ids.SceneID(targetSceneID), CharacterID: ids.CharacterID(characterID)}
	addJSON, _ := json.Marshal(addPayload)
	addEvt := command.NewEvent(cmd, EventTypeCharacterAdded, "scene", targetSceneID, addJSON, ts)

	return command.Accept(removeEvt, addEvt)
}

func decideTransition(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload TransitionPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sourceSceneID := strings.TrimSpace(payload.SourceSceneID.String())
	if sourceSceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSourceSceneIDRequired, Message: "source scene id is required"})
	}
	newSceneID := strings.TrimSpace(payload.NewSceneID.String())
	if newSceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeNewSceneIDRequired, Message: "new scene id is required"})
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneNameRequired, Message: "scene name is required"})
	}
	if rejection := requireActiveScene(scenes, sourceSceneID); rejection != nil {
		return command.Reject(*rejection)
	}

	source := scenes[ids.SceneID(sourceSceneID)]
	charIDs := sortedCharacterIDs(source.Characters)
	description := strings.TrimSpace(payload.Description)

	ts := now().UTC()
	events := make([]event.Event, 0, 2+len(charIDs))

	// 1. Create the new scene.
	createPayload := CreatePayload{
		SceneID:      ids.SceneID(newSceneID),
		Name:         name,
		Description:  description,
		CharacterIDs: charIDs,
	}
	createJSON, _ := json.Marshal(createPayload)
	events = append(events, command.NewEvent(cmd, EventTypeCreated, "scene", newSceneID, createJSON, ts))

	// 2. Add each character to the new scene.
	for _, charID := range charIDs {
		addPayload := CharacterAddedPayload{SceneID: ids.SceneID(newSceneID), CharacterID: charID}
		addJSON, _ := json.Marshal(addPayload)
		events = append(events, command.NewEvent(cmd, EventTypeCharacterAdded, "scene", newSceneID, addJSON, ts))
	}

	// 3. End the source scene.
	endPayload := EndPayload{SceneID: ids.SceneID(sourceSceneID), Reason: "transitioned"}
	endJSON, _ := json.Marshal(endPayload)
	events = append(events, command.NewEvent(cmd, EventTypeEnded, "scene", sourceSceneID, endJSON, ts))

	return command.Accept(events...)
}

func decideGateOpen(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload GateOpenedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneGateIDRequired, Message: "gate id is required"})
	}
	gateType, err := NormalizeGateType(payload.GateType)
	if err != nil {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneGateTypeRequired, Message: err.Error()})
	}
	scene := scenes[ids.SceneID(sceneID)]
	if scene.GateOpen {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneGateAlreadyOpen, Message: "scene gate is already open"})
	}

	normalized := GateOpenedPayload{
		SceneID:  ids.SceneID(sceneID),
		GateID:   ids.GateID(gateID),
		GateType: gateType,
		Reason:   strings.TrimSpace(payload.Reason),
		Metadata: payload.Metadata,
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeGateOpened, "scene_gate", gateID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideGateResolve(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload GateResolvedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneGateIDRequired, Message: "gate id is required"})
	}
	scene := scenes[ids.SceneID(sceneID)]
	if !scene.GateOpen {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneGateNotOpen, Message: "scene gate is not open"})
	}

	normalized := GateResolvedPayload{
		SceneID:    ids.SceneID(sceneID),
		GateID:     ids.GateID(gateID),
		Decision:   strings.TrimSpace(payload.Decision),
		Resolution: payload.Resolution,
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeGateResolved, "scene_gate", gateID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideGateAbandon(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload GateAbandonedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneGateIDRequired, Message: "gate id is required"})
	}
	scene := scenes[ids.SceneID(sceneID)]
	if !scene.GateOpen {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneGateNotOpen, Message: "scene gate is not open"})
	}

	normalized := GateAbandonedPayload{
		SceneID: ids.SceneID(sceneID),
		GateID:  ids.GateID(gateID),
		Reason:  strings.TrimSpace(payload.Reason),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeGateAbandoned, "scene_gate", gateID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideSpotlightSet(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload SpotlightSetPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	spotlightType := SpotlightType(strings.TrimSpace(string(payload.SpotlightType)))
	if spotlightType == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSpotlightTypeRequired, Message: "spotlight type is required"})
	}
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if spotlightType == SpotlightTypeCharacter {
		if characterID == "" {
			return command.Reject(command.Rejection{Code: rejectionCodeCharacterIDRequired, Message: "character id is required for character spotlight"})
		}
		scene := scenes[ids.SceneID(sceneID)]
		if !scene.Characters[ids.CharacterID(characterID)] {
			return command.Reject(command.Rejection{Code: rejectionCodeCharacterNotInScene, Message: "character is not in scene"})
		}
	}

	normalized := SpotlightSetPayload{SceneID: ids.SceneID(sceneID), SpotlightType: spotlightType, CharacterID: ids.CharacterID(characterID)}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeSpotlightSet, "scene", sceneID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideSpotlightClear(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload SpotlightClearedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return rejectPayloadDecode(cmd, err)
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSceneIDRequired, Message: "scene id is required"})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	scene := scenes[ids.SceneID(sceneID)]
	if scene.SpotlightType == "" {
		return command.Reject(command.Rejection{Code: rejectionCodeSpotlightNotSet, Message: "spotlight is not set"})
	}

	normalized := SpotlightClearedPayload{SceneID: ids.SceneID(sceneID), Reason: strings.TrimSpace(payload.Reason)}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeSpotlightCleared, "scene", sceneID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

// requireActiveScene validates that the scene exists and is active.
func requireActiveScene(scenes map[ids.SceneID]State, sceneID string) *command.Rejection {
	scene, ok := scenes[ids.SceneID(sceneID)]
	if !ok {
		return &command.Rejection{Code: rejectionCodeSceneNotFound, Message: "scene not found"}
	}
	if !scene.Active {
		return &command.Rejection{Code: rejectionCodeSceneNotActive, Message: "scene is not active"}
	}
	return nil
}

func rejectPayloadDecode(cmd command.Command, err error) command.Decision {
	return command.Reject(command.Rejection{
		Code:    command.RejectionCodePayloadDecodeFailed,
		Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
	})
}

// normalizeCharacterIDs trims, deduplicates, and filters empty character IDs.
func normalizeCharacterIDs(charIDs []ids.CharacterID) []ids.CharacterID {
	seen := make(map[ids.CharacterID]bool, len(charIDs))
	result := make([]ids.CharacterID, 0, len(charIDs))
	for _, id := range charIDs {
		trimmed := ids.CharacterID(strings.TrimSpace(id.String()))
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

// sortedCharacterIDs returns a stable-sorted slice of character IDs from a map.
func sortedCharacterIDs(chars map[ids.CharacterID]bool) []ids.CharacterID {
	strs := make([]string, 0, len(chars))
	for id := range chars {
		strs = append(strs, string(id))
	}
	// Sort for deterministic event order in replay.
	slices.Sort(strs)
	result := make([]ids.CharacterID, 0, len(strs))
	for _, s := range strs {
		result = append(result, ids.CharacterID(s))
	}
	return result
}

// sortStrings sorts a slice of strings in place.
func sortStrings(s []string) {
	slices.Sort(s)
}
