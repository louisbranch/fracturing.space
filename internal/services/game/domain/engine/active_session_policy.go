package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

const (
	// RejectionCodeCampaignActiveSessionLocked identifies active-session lock rejections.
	RejectionCodeCampaignActiveSessionLocked = "CAMPAIGN_ACTIVE_SESSION_LOCKED"
)

// ActiveSessionCommandPolicy classifies command behavior while a session is active.
type ActiveSessionCommandPolicy string

const (
	// ActiveSessionCommandPolicyBlocked rejects command execution while active.
	ActiveSessionCommandPolicyBlocked ActiveSessionCommandPolicy = "blocked"
	// ActiveSessionCommandPolicyAllowed permits command execution while active.
	ActiveSessionCommandPolicyAllowed ActiveSessionCommandPolicy = "allowed"
)

// ActiveSessionPolicyForCommandType returns how a command behaves while a
// campaign session is active.
//
// Policy is intentionally namespace-driven so new command families must be
// reviewed and classified once, then inherited by all commands in that family.
func ActiveSessionPolicyForCommandType(cmdType command.Type) (ActiveSessionCommandPolicy, bool) {
	switch cmdType {
	case commandids.DaggerheartCharacterProfileReplace, commandids.DaggerheartCharacterProfileDelete:
		return ActiveSessionCommandPolicyBlocked, true
	}
	switch commandNamespace(cmdType) {
	case "campaign", "participant", "invite", "character":
		return ActiveSessionCommandPolicyBlocked, true
	case "session", "scene", "action", "story", "sys":
		return ActiveSessionCommandPolicyAllowed, true
	default:
		return "", false
	}
}

// ActiveSessionPolicyForCommand returns how a specific command behaves while a
// campaign session is active.
//
// Family-level policy remains centralized via ActiveSessionPolicyForCommandType.
// Command-level context then narrows exceptions for in-game mutations.
func ActiveSessionPolicyForCommand(cmd command.Command) (ActiveSessionCommandPolicy, bool) {
	basePolicy, ok := ActiveSessionPolicyForCommandType(cmd.Type)
	if !ok {
		return "", false
	}
	if basePolicy == ActiveSessionCommandPolicyBlocked && commandNamespace(cmd.Type) == "character" && isInGameCharacterCommand(cmd) {
		return ActiveSessionCommandPolicyAllowed, true
	}
	return basePolicy, true
}

// RejectActiveSessionBlockedCommand returns a rejection when active-session
// policy blocks a command.
func RejectActiveSessionBlockedCommand(state session.State, cmd command.Command) (command.Decision, bool) {
	if !state.Started {
		return command.Decision{}, false
	}
	policy, ok := ActiveSessionPolicyForCommand(cmd)
	if !ok || policy != ActiveSessionCommandPolicyBlocked {
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

func commandNamespace(cmdType command.Type) string {
	trimmed := strings.TrimSpace(string(cmdType))
	if trimmed == "" {
		return ""
	}
	dot := strings.Index(trimmed, ".")
	if dot <= 0 {
		return trimmed
	}
	return trimmed[:dot]
}
