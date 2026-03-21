package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// decideOOCPause opens the out-of-character coordination surface for the session.
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
	payload.RequestedByParticipantID = ids.ParticipantID(strings.TrimSpace(payload.RequestedByParticipantID.String()))
	payload.Reason = strings.TrimSpace(payload.Reason)
	payload.InterruptedSceneID = ids.SceneID(strings.TrimSpace(payload.InterruptedSceneID.String()))
	payload.InterruptedPhaseID = strings.TrimSpace(payload.InterruptedPhaseID)
	payload.InterruptedPhaseStatus = strings.TrimSpace(payload.InterruptedPhaseStatus)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeOOCPaused, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}

// decideOOCPost records one OOC discussion post while the session is paused.
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

// decideOOCReadyMark records one participant's readiness to resume play.
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

// decideOOCReadyClear removes one participant's readiness mark while paused.
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

// decideOOCResume closes the OOC pause and returns the session to play.
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

// decideOOCInterruptionResolve clears the pending GM resolution gate after OOC.
func decideOOCInterruptionResolve(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.OOCPaused || !state.OOCResolutionPending {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionOOCResolutionNotPending,
			Message: "session is not waiting for post-ooc resolution",
		})
	}
	var payload OOCInterruptionResolvedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	payload.SessionID = ids.SessionID(cmd.SessionID)
	payload.Resolution = strings.TrimSpace(payload.Resolution)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeOOCInterruptionResolved, "session", cmd.SessionID.String(), payloadJSON, now().UTC())
	evt.SessionID = cmd.SessionID
	return command.Accept(evt)
}
