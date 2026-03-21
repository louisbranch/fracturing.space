package scene

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideGMInteractionCommit(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload GMInteractionCommittedPayload
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
			Code:    rejectionCodeSceneGMInteractionParticipantRequired,
			Message: "participant id is required",
		})
	}
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSceneGMInteractionTitleRequired,
			Message: "gm interaction title is required",
		})
	}
	beats, rejection := normalizeGMInteractionBeats(payload.Beats)
	if rejection != nil {
		return command.Reject(*rejection)
	}
	payload.SceneID = ids.SceneID(sceneID)
	payload.Title = title
	payload.InteractionID = strings.TrimSpace(payload.InteractionID)
	payload.PhaseID = strings.TrimSpace(payload.PhaseID)
	payload.CharacterIDs = normalizeCharacterIDs(payload.CharacterIDs)
	payload.Beats = beats
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeGMInteractionCommitted, "scene", sceneID, payloadJSON, now().UTC())
	evt.SceneID = ids.SceneID(sceneID)
	return command.Accept(evt)
}
