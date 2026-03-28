package campaign

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

func decideAIBind(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}
	var payload AIBindPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: commandDecodeMessage(cmd, err),
		})
	}
	agentID := strings.TrimSpace(payload.AIAgentID)
	if agentID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignAIAgentIDRequired,
			Message: "ai agent id is required",
		})
	}

	normalizedPayload := AIBindPayload{AIAgentID: agentID}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeAIBound, "campaign", string(cmd.CampaignID), payloadJSON, command.RequireNowFunc(now)().UTC())
	return command.Accept(evt)
}

func decideAIUnbind(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}
	payloadJSON, _ := json.Marshal(AIUnbindPayload{})
	evt := command.NewEvent(cmd, EventTypeAIUnbound, "campaign", string(cmd.CampaignID), payloadJSON, command.RequireNowFunc(now)().UTC())
	return command.Accept(evt)
}

func decideAIAuthRotate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}

	var payload AIAuthRotatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: commandDecodeMessage(cmd, err),
		})
	}

	payload.EpochAfter = state.AIAuthEpoch + 1
	payload.Reason = strings.TrimSpace(payload.Reason)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeAIAuthRotated, "campaign", string(cmd.CampaignID), payloadJSON, command.RequireNowFunc(now)().UTC())
	return command.Accept(evt)
}
