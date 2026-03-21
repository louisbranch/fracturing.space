package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// decideSceneActivated routes the in-character interaction surface to one scene.
func decideSceneActivated(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload SceneActivatedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	activeSceneID := strings.TrimSpace(payload.ActiveSceneID.String())
	if activeSceneID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionActiveSceneRequired,
			Message: "active scene id is required",
		})
	}
	if ids.SceneID(activeSceneID) == state.ActiveSceneID {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionActiveSceneUnchanged,
			Message: "active scene is already set",
		})
	}

	normalized := SceneActivatedPayload{
		SessionID:     ids.SessionID(cmd.SessionID),
		ActiveSceneID: ids.SceneID(activeSceneID),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeSceneActivated, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

// decideGMAuthoritySet records which participant currently holds GM authority.
func decideGMAuthoritySet(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload GMAuthoritySetPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionGMAuthorityRequired,
			Message: "gm authority participant id is required",
		})
	}
	if ids.ParticipantID(participantID) == state.GMAuthorityParticipantID {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionGMAuthorityUnchanged,
			Message: "gm authority participant is already set",
		})
	}
	normalized := GMAuthoritySetPayload{
		SessionID:     ids.SessionID(cmd.SessionID),
		ParticipantID: ids.ParticipantID(participantID),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeGMAuthoritySet, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}
