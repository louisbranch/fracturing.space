package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideActiveSceneSet(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload ActiveSceneSetPayload
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

	normalized := ActiveSceneSetPayload{
		SessionID:     ids.SessionID(cmd.SessionID),
		ActiveSceneID: ids.SceneID(activeSceneID),
	}
	payloadJSON, _ := json.Marshal(normalized)
	evt := command.NewEvent(cmd, EventTypeActiveSceneSet, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

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

func decideOOCPause(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.OOCPaused {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCAlreadyOpen,
			Message: "session is already paused for out-of-character discussion",
		})
	}
	var payload OOCPausedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.Reason = strings.TrimSpace(payload.Reason)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeOOCPaused, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

func decideOOCPost(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.OOCPaused {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCNotOpen,
			Message: "session is not paused for out-of-character discussion",
		})
	}
	var payload OOCPostedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	if strings.TrimSpace(payload.PostID) == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCPostIDRequired,
			Message: "ooc post id is required",
		})
	}
	if strings.TrimSpace(payload.ParticipantID.String()) == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCParticipantRequired,
			Message: "participant id is required",
		})
	}
	body := strings.TrimSpace(payload.Body)
	if body == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCBodyRequired,
			Message: "ooc body is required",
		})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.Body = body
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeOOCPosted, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

func decideOOCReadyMark(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.OOCPaused {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCNotOpen,
			Message: "session is not paused for out-of-character discussion",
		})
	}
	var payload OOCReadyMarkedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCParticipantRequired,
			Message: "participant id is required",
		})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.ParticipantID = ids.ParticipantID(participantID)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeOOCReadyMarked, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

func decideOOCReadyClear(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.OOCPaused {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCNotOpen,
			Message: "session is not paused for out-of-character discussion",
		})
	}
	var payload OOCReadyClearedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCParticipantRequired,
			Message: "participant id is required",
		})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.ParticipantID = ids.ParticipantID(participantID)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeOOCReadyCleared, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

func decideOOCResume(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.OOCPaused {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCNotOpen,
			Message: "session is not paused for out-of-character discussion",
		})
	}
	var payload OOCResumedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.Reason = strings.TrimSpace(payload.Reason)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeOOCResumed, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

func decideAITurnQueue(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload AITurnQueuedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	turnToken := strings.TrimSpace(payload.TurnToken)
	if turnToken == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnTokenRequired,
			Message: "ai turn token is required",
		})
	}
	ownerParticipantID := strings.TrimSpace(payload.OwnerParticipantID.String())
	if ownerParticipantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnOwnerRequired,
			Message: "ai turn owner participant id is required",
		})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.TurnToken = turnToken
	payload.OwnerParticipantID = ids.ParticipantID(ownerParticipantID)
	payload.SourceEventType = strings.TrimSpace(payload.SourceEventType)
	payload.SourceSceneID = ids.SceneID(strings.TrimSpace(payload.SourceSceneID.String()))
	payload.SourcePhaseID = strings.TrimSpace(payload.SourcePhaseID)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeAITurnQueued, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

func decideAITurnStart(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload AITurnRunningPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	turnToken := strings.TrimSpace(payload.TurnToken)
	if turnToken == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnTokenRequired,
			Message: "ai turn token is required",
		})
	}
	if state.AITurnStatus != AITurnStatusQueued {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnNotQueued,
			Message: "ai turn is not queued",
		})
	}
	if turnToken != strings.TrimSpace(state.AITurnToken) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnTokenMismatch,
			Message: "ai turn token does not match the queued turn",
		})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.TurnToken = turnToken
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeAITurnRunning, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

func decideAITurnFail(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload AITurnFailedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	turnToken := strings.TrimSpace(payload.TurnToken)
	if turnToken == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnTokenRequired,
			Message: "ai turn token is required",
		})
	}
	if state.AITurnStatus != AITurnStatusQueued && state.AITurnStatus != AITurnStatusRunning {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnNotActive,
			Message: "ai turn is not active",
		})
	}
	if turnToken != strings.TrimSpace(state.AITurnToken) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnTokenMismatch,
			Message: "ai turn token does not match the active turn",
		})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.TurnToken = turnToken
	payload.LastError = strings.TrimSpace(payload.LastError)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeAITurnFailed, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

func decideAITurnClear(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload AITurnClearedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	turnToken := strings.TrimSpace(payload.TurnToken)
	if strings.TrimSpace(state.AITurnToken) != "" && turnToken != "" && turnToken != strings.TrimSpace(state.AITurnToken) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnTokenMismatch,
			Message: "ai turn token does not match the active turn",
		})
	}
	if state.AITurnStatus == "" && strings.TrimSpace(state.AITurnToken) == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionAITurnNotActive,
			Message: "ai turn is not active",
		})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.TurnToken = turnToken
	payload.Reason = strings.TrimSpace(payload.Reason)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeAITurnCleared, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}
