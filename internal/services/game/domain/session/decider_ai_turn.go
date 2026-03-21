package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// decideAITurnQueue records a queued AI turn owned by the current GM authority.
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

// decideAITurnStart transitions a queued AI turn into its running state.
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

// decideAITurnFail records a retryable orchestration failure for the active AI turn.
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

// decideAITurnClear clears the current AI turn and records the orchestration reason.
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
