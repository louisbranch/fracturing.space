package scene

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideGMOutputCommit(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload GMOutputCommittedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		sceneID = strings.TrimSpace(cmd.SceneID.String())
	}
	if sceneID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSceneIDRequired,
			Message: "scene id is required",
		})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	if strings.TrimSpace(payload.ParticipantID.String()) == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSceneGMOutputParticipantRequired,
			Message: "participant id is required",
		})
	}
	text := strings.TrimSpace(payload.Text)
	if text == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSceneGMOutputRequired,
			Message: "gm output text is required",
		})
	}
	payload.SceneID = ids.SceneID(sceneID)
	payload.Text = text
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeGMOutputCommitted, "scene", sceneID, payloadJSON, now().UTC())
	evt.SceneID = ids.SceneID(sceneID)
	return command.Accept(evt)
}
