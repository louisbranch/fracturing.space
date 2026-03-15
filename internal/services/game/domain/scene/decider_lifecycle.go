package scene

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

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

	createPayload := CreatePayload{
		SceneID:      ids.SceneID(newSceneID),
		Name:         name,
		Description:  description,
		CharacterIDs: charIDs,
	}
	createJSON, _ := json.Marshal(createPayload)
	events = append(events, command.NewEvent(cmd, EventTypeCreated, "scene", newSceneID, createJSON, ts))

	for _, charID := range charIDs {
		addPayload := CharacterAddedPayload{SceneID: ids.SceneID(newSceneID), CharacterID: charID}
		addJSON, _ := json.Marshal(addPayload)
		events = append(events, command.NewEvent(cmd, EventTypeCharacterAdded, "scene", newSceneID, addJSON, ts))
	}

	endPayload := EndPayload{SceneID: ids.SceneID(sourceSceneID), Reason: "transitioned"}
	endJSON, _ := json.Marshal(endPayload)
	events = append(events, command.NewEvent(cmd, EventTypeEnded, "scene", sourceSceneID, endJSON, ts))

	return command.Accept(events...)
}
