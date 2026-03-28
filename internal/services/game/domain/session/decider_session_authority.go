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

// decideCharacterControllerSet records one session-scoped controller assignment.
func decideCharacterControllerSet(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload CharacterControllerSetPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionCharacterRequired,
			Message: "character id is required",
		})
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionCharacterControllerRequired,
			Message: "character controller participant id is required",
		})
	}
	if state.CharacterControllers != nil && state.CharacterControllers[ids.CharacterID(characterID)] == ids.ParticipantID(participantID) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionCharacterControllerUnchanged,
			Message: "character controller participant is already set",
		})
	}
	normalized := CharacterControllerSetPayload{
		SessionID:     ids.SessionID(cmd.SessionID),
		CharacterID:   ids.CharacterID(characterID),
		ParticipantID: ids.ParticipantID(participantID),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeCharacterControllerSet, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}
