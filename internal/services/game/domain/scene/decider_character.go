package scene

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

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
