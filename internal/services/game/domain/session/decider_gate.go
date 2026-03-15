package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideGateOpen(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload GateOpenedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	gateID := strings.TrimSpace(payload.GateID.String())
	gateType, err := NormalizeGateType(payload.GateType)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionGateTypeRequired,
			Message: err.Error(),
		})
	}
	reason := strings.TrimSpace(payload.Reason)
	if gateID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionGateIDRequired,
			Message: "gate id is required",
		})
	}
	if state.GateOpen {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionGateAlreadyOpen,
			Message: "session gate is already open",
		})
	}
	metadata, err := NormalizeGateWorkflowMetadata(gateType, payload.Metadata)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSessionGateMetadataInvalid,
			Message: err.Error(),
		})
	}

	normalizedPayload := GateOpenedPayload{GateID: ids.GateID(gateID), GateType: gateType, Reason: reason, Metadata: metadata}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeGateOpened, "session_gate", gateID, payloadJSON, now().UTC())

	return command.Accept(evt)
}

func decideGateResolve(state State, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(
		cmd,
		EventTypeGateResolved,
		"session_gate",
		func(payload *GateResolvedPayload) string {
			return payload.GateID.String()
		},
		func(payload *GateResolvedPayload, _ func() time.Time) *command.Rejection {
			payload.GateID = ids.GateID(strings.TrimSpace(payload.GateID.String()))
			payload.Decision = strings.TrimSpace(payload.Decision)
			if !state.GateOpen {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateNotOpen,
					Message: "session gate is not open",
				}
			}
			if payload.GateID == "" {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateIDRequired,
					Message: "gate id is required",
				}
			}
			if state.GateID != "" && payload.GateID != state.GateID {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateMismatch,
					Message: "gate id does not match the active session gate",
				}
			}
			return nil
		},
		now,
	)
}

func decideGateRespond(state State, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(
		cmd,
		EventTypeGateResponseRecorded,
		"session_gate",
		func(payload *GateResponseRecordedPayload) string {
			return payload.GateID.String()
		},
		func(payload *GateResponseRecordedPayload, _ func() time.Time) *command.Rejection {
			payload.GateID = ids.GateID(strings.TrimSpace(payload.GateID.String()))
			payload.ParticipantID = ids.ParticipantID(strings.TrimSpace(payload.ParticipantID.String()))
			if !state.GateOpen {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateNotOpen,
					Message: "session gate is not open",
				}
			}
			if payload.GateID == "" {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateIDRequired,
					Message: "gate id is required",
				}
			}
			if state.GateID != "" && payload.GateID != state.GateID {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateMismatch,
					Message: "gate id does not match the active session gate",
				}
			}
			if payload.ParticipantID == "" {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateParticipantRequired,
					Message: "participant id is required",
				}
			}
			decision, response, err := ValidateGateResponse(
				state.GateType,
				state.GateMetadataJSON,
				payload.ParticipantID.String(),
				payload.Decision,
				payload.Response,
			)
			if err != nil {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateResponseInvalid,
					Message: err.Error(),
				}
			}
			payload.Decision = decision
			payload.Response = response
			return nil
		},
		now,
	)
}

func decideGateAbandon(state State, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(
		cmd,
		EventTypeGateAbandoned,
		"session_gate",
		func(payload *GateAbandonedPayload) string {
			return payload.GateID.String()
		},
		func(payload *GateAbandonedPayload, _ func() time.Time) *command.Rejection {
			payload.GateID = ids.GateID(strings.TrimSpace(payload.GateID.String()))
			payload.Reason = strings.TrimSpace(payload.Reason)
			if !state.GateOpen {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateNotOpen,
					Message: "session gate is not open",
				}
			}
			if payload.GateID == "" {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateIDRequired,
					Message: "gate id is required",
				}
			}
			if state.GateID != "" && payload.GateID != state.GateID {
				return &command.Rejection{
					Code:    rejectionCodeSessionGateMismatch,
					Message: "gate id does not match the active session gate",
				}
			}
			return nil
		},
		now,
	)
}
