package participant

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideBind(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)

	if rejection, ok := ensureParticipantActive(state); !ok {
		return command.Reject(rejection)
	}
	if isAIController(string(state.Controller)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantAIIdentityLocked,
			Message: "ai-controlled participants cannot change user identity bindings",
		})
	}
	payload, err := decodeCommandPayload[BindPayload](cmd)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantIDRequired,
			Message: "participant id is required",
		})
	}
	userID := strings.TrimSpace(payload.UserID.String())
	if userID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantUserIDRequired,
			Message: "user id is required",
		})
	}
	if strings.TrimSpace(state.UserID.String()) != "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantAlreadyClaimed,
			Message: "participant already claimed",
		})
	}

	normalizedPayload := BindPayload{ParticipantID: ids.ParticipantID(participantID), UserID: ids.UserID(userID)}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeBound, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideUnbind(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)

	if rejection, ok := ensureParticipantActive(state); !ok {
		return command.Reject(rejection)
	}
	if isAIController(string(state.Controller)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantAIIdentityLocked,
			Message: "ai-controlled participants cannot change user identity bindings",
		})
	}
	payload, err := decodeCommandPayload[UnbindPayload](cmd)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantIDRequired,
			Message: "participant id is required",
		})
	}
	userID := strings.TrimSpace(payload.UserID.String())
	if userID != "" && userID != strings.TrimSpace(string(state.UserID)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantUserIDMismatch,
			Message: "participant user id mismatch",
		})
	}
	reason := strings.TrimSpace(payload.Reason)

	normalizedPayload := UnbindPayload{ParticipantID: ids.ParticipantID(participantID), UserID: ids.UserID(userID), Reason: reason}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeUnbound, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}

func decideSeatReassign(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)

	if rejection, ok := ensureParticipantActive(state); !ok {
		return command.Reject(rejection)
	}
	if isAIController(string(state.Controller)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantAIIdentityLocked,
			Message: "ai-controlled participants cannot change user identity bindings",
		})
	}
	payload, err := decodeCommandPayload[SeatReassignPayload](cmd)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantIDRequired,
			Message: "participant id is required",
		})
	}
	priorUserID := strings.TrimSpace(payload.PriorUserID.String())
	if priorUserID != "" && priorUserID != strings.TrimSpace(string(state.UserID)) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantUserIDMismatch,
			Message: "participant user id mismatch",
		})
	}
	userID := strings.TrimSpace(payload.UserID.String())
	if userID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantUserIDRequired,
			Message: "user id is required",
		})
	}
	reason := strings.TrimSpace(payload.Reason)

	normalizedPayload := SeatReassignPayload{
		ParticipantID: ids.ParticipantID(participantID),
		PriorUserID:   ids.UserID(priorUserID),
		UserID:        ids.UserID(userID),
		Reason:        reason,
	}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeSeatReassigned, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}
