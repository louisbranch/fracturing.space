package scene

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

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
