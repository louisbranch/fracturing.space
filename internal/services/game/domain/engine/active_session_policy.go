package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

const (
	// RejectionCodeCampaignActiveSessionLocked identifies active-session lock rejections.
	RejectionCodeCampaignActiveSessionLocked = "CAMPAIGN_ACTIVE_SESSION_LOCKED"
)

// ActiveSessionPolicyForDefinition resolves how a command behaves while a
// campaign session is active using its registered command metadata.
func ActiveSessionPolicyForDefinition(
	definition command.Definition,
	cmd command.Command,
) (command.ActiveSessionClassification, bool) {
	classification := definition.ActiveSession.Classification
	if classification == "" {
		return "", false
	}
	if classification == command.ActiveSessionClassificationBlocked &&
		definition.ActiveSession.AllowInGameSystemActor &&
		isInGameCharacterCommand(cmd) {
		return command.ActiveSessionClassificationAllowed, true
	}
	return classification, true
}

// RejectActiveSessionBlockedCommand returns a rejection when active-session
// policy blocks a command.
func RejectActiveSessionBlockedCommand(
	state session.State,
	cmd command.Command,
	definition command.Definition,
) (command.Decision, bool) {
	if !state.Started {
		return command.Decision{}, false
	}
	policy, ok := ActiveSessionPolicyForDefinition(definition, cmd)
	if !ok || policy != command.ActiveSessionClassificationBlocked {
		return command.Decision{}, false
	}
	message := "campaign has an active session"
	if sessionID := strings.TrimSpace(string(state.SessionID)); sessionID != "" {
		message = fmt.Sprintf("campaign has an active session: active_session_id=%s", sessionID)
	}
	return command.Reject(command.Rejection{
		Code:    RejectionCodeCampaignActiveSessionLocked,
		Message: message,
	}), true
}

func isInGameCharacterCommand(cmd command.Command) bool {
	return cmd.ActorType == command.ActorTypeSystem && strings.TrimSpace(cmd.SessionID.String()) != ""
}
