package action

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func decideOutcomeApply(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload OutcomeApplyPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRequestIDRequired,
			Message: "request_id is required",
		})
	}
	if payload.RollSeq == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRollSeqRequired,
			Message: "roll_seq must be greater than zero",
		})
	}
	if hasSystemOwnedOutcomeEffect(payload.PreEffects) || hasSystemOwnedOutcomeEffect(payload.PostEffects) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeOutcomeEffectSystemOwnedForbidden,
			Message: "core action.outcome.apply cannot emit system-owned effects",
		})
	}
	if hasDisallowedCoreOutcomeEffect(payload.PreEffects) || hasDisallowedCoreOutcomeEffect(payload.PostEffects) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeOutcomeEffectTypeForbidden,
			Message: "core action.outcome.apply effect type is not allowed",
		})
	}
	if _, alreadyApplied := state.AppliedOutcomes[payload.RollSeq]; alreadyApplied {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeOutcomeAlreadyApplied,
			Message: ErrOutcomeAlreadyApplied.Error(),
		})
	}

	events := make([]event.Event, 0, len(payload.PreEffects)+len(payload.PostEffects)+1)
	for _, effect := range payload.PreEffects {
		events = append(events, buildOutcomeEffectEvent(cmd, now, effect))
	}

	postEffects := payload.PostEffects
	payload.PreEffects = nil
	payload.PostEffects = nil
	events = append(events, acceptActionEvent(cmd, now, EventTypeOutcomeApplied, "outcome", requestID, payload).Events...)

	for _, effect := range postEffects {
		events = append(events, buildOutcomeEffectEvent(cmd, now, effect))
	}
	return command.Accept(events...)
}

func decideOutcomeReject(cmd command.Command, now func() time.Time) command.Decision {
	var payload OutcomeRejectPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRequestIDRequired,
			Message: "request_id is required",
		})
	}
	if payload.RollSeq == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRollSeqRequired,
			Message: "roll_seq must be greater than zero",
		})
	}
	return acceptActionEvent(cmd, now, EventTypeOutcomeRejected, "outcome", requestID, payload)
}
