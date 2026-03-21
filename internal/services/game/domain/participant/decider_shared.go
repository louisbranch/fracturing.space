package participant

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func ensureParticipantActive(state State) (command.Rejection, bool) {
	if !state.Joined || state.Left {
		return command.Rejection{
			Code:    rejectionCodeParticipantNotJoined,
			Message: "participant not joined",
		}, false
	}
	return command.Rejection{}, true
}

func decodeCommandPayload[T any](cmd command.Command) (T, error) {
	var payload T
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func validateAISeatInvariant(userID, role, controller, access string) (command.Rejection, bool) {
	normalizedController, ok := normalizeControllerLabel(controller)
	if !ok || normalizedController != "ai" {
		return command.Rejection{}, true
	}
	normalizedRole, ok := normalizeRoleLabel(role)
	if !ok || normalizedRole != "gm" {
		return command.Rejection{
			Code:    rejectionCodeParticipantAIRoleRequired,
			Message: "ai-controlled participants must use gm role",
		}, false
	}
	normalizedAccess, ok := normalizeCampaignAccessLabel(access)
	if !ok || normalizedAccess != "manager" {
		return command.Rejection{
			Code:    rejectionCodeParticipantAIAccessRequired,
			Message: "ai-controlled participants must use manager campaign access",
		}, false
	}
	if strings.TrimSpace(userID) != "" {
		return command.Rejection{
			Code:    rejectionCodeParticipantAIUserIDForbidden,
			Message: "ai-controlled participants must not have a user id",
		}, false
	}
	return command.Rejection{}, true
}

func isAIController(controller string) bool {
	normalized, ok := normalizeControllerLabel(controller)
	return ok && normalized == "ai"
}

// normalizeRoleLabel returns a canonical role label.
func normalizeRoleLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "GM", "ROLE_GM", "PARTICIPANT_ROLE_GM":
		return "gm", true
	case "PLAYER", "ROLE_PLAYER", "PARTICIPANT_ROLE_PLAYER":
		return "player", true
	default:
		return "", false
	}
}

// normalizeControllerLabel returns a canonical controller label.
func normalizeControllerLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "HUMAN", "CONTROLLER_HUMAN":
		return "human", true
	case "AI", "CONTROLLER_AI":
		return "ai", true
	default:
		return "", false
	}
}

// normalizeCampaignAccessLabel returns a canonical access label.
func normalizeCampaignAccessLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "MEMBER", "CAMPAIGN_ACCESS_MEMBER":
		return "member", true
	case "MANAGER", "CAMPAIGN_ACCESS_MANAGER":
		return "manager", true
	case "OWNER", "CAMPAIGN_ACCESS_OWNER":
		return "owner", true
	default:
		return "", false
	}
}

// acceptParticipantEvent creates the standard participant event envelope for
// accepted commands. Centralizing this constructor keeps participant event
// metadata consistent across all participant command handlers.
func acceptParticipantEvent(cmd command.Command, now func() time.Time, eventType event.Type, participantID string, payload any) command.Decision {
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, eventType, "participant", participantID, payloadJSON, now().UTC())
	return command.Accept(evt)
}
