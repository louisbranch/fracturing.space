package scene

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

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
