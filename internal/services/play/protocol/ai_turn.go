package protocol

import (
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// AITurnState represents the current AI GM turn state.
type AITurnState struct {
	Status             string `json:"status,omitempty"`
	OwnerParticipantID string `json:"owner_participant_id,omitempty"`
	LastError          string `json:"last_error,omitempty"`
}

// AITurnFromGameAITurn maps a proto AITurnState to protocol.
func AITurnFromGameAITurn(aiTurn *gamev1.AITurnState) *AITurnState {
	if aiTurn == nil {
		return nil
	}
	status := aiTurnStatusString(aiTurn.GetStatus())
	owner := strings.TrimSpace(aiTurn.GetOwnerParticipantId())
	lastErr := strings.TrimSpace(aiTurn.GetLastError())
	if status == "" && owner == "" && lastErr == "" {
		return nil
	}
	return &AITurnState{
		Status:             status,
		OwnerParticipantID: owner,
		LastError:          lastErr,
	}
}

func aiTurnStatusString(value gamev1.AITurnStatus) string {
	return ProtoEnumToLower(value, gamev1.AITurnStatus_AI_TURN_STATUS_UNSPECIFIED, "AI_TURN_STATUS_")
}
