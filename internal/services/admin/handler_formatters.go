package admin

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"golang.org/x/text/message"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// formatGmMode returns a display label for a GM mode enum.
func formatGmMode(mode statev1.GmMode, loc *message.Printer) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return loc.Sprintf("label.human")
	case statev1.GmMode_AI:
		return loc.Sprintf("label.ai")
	case statev1.GmMode_HYBRID:
		return loc.Sprintf("label.hybrid")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatGameSystem(system commonv1.GameSystem, loc *message.Printer) string {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return loc.Sprintf("label.daggerheart")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatImplementationStage(stage commonv1.GameSystemImplementationStage, loc *message.Printer) string {
	switch stage {
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PLANNED:
		return loc.Sprintf("label.system_stage_planned")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL:
		return loc.Sprintf("label.system_stage_partial")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_COMPLETE:
		return loc.Sprintf("label.system_stage_complete")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_DEPRECATED:
		return loc.Sprintf("label.system_stage_deprecated")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatOperationalStatus(status commonv1.GameSystemOperationalStatus, loc *message.Printer) string {
	switch status {
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OFFLINE:
		return loc.Sprintf("label.system_status_offline")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_DEGRADED:
		return loc.Sprintf("label.system_status_degraded")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL:
		return loc.Sprintf("label.system_status_operational")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_MAINTENANCE:
		return loc.Sprintf("label.system_status_maintenance")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatAccessLevel(level commonv1.GameSystemAccessLevel, loc *message.Printer) string {
	switch level {
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_INTERNAL:
		return loc.Sprintf("label.system_access_internal")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA:
		return loc.Sprintf("label.system_access_beta")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_PUBLIC:
		return loc.Sprintf("label.system_access_public")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_RETIRED:
		return loc.Sprintf("label.system_access_retired")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func parseSystemID(value string) commonv1.GameSystem {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
	if trimmed == "DAGGERHEART" {
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	}
	if enumValue, ok := commonv1.GameSystem_value[trimmed]; ok {
		return commonv1.GameSystem(enumValue)
	}
	return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
}

func parseGameSystem(value string) (commonv1.GameSystem, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "daggerheart":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, true
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, false
	}
}

func parseGmMode(value string) (statev1.GmMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human":
		return statev1.GmMode_HUMAN, true
	case "ai":
		return statev1.GmMode_AI, true
	case "hybrid":
		return statev1.GmMode_HYBRID, true
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED, false
	}
}

// formatSessionStatus returns a display label for a session status.
func formatSessionStatus(status statev1.SessionStatus, loc *message.Printer) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return loc.Sprintf("label.active")
	case statev1.SessionStatus_SESSION_ENDED:
		return loc.Sprintf("label.ended")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatInviteStatus(status statev1.InviteStatus, loc *message.Printer) (string, string) {
	switch status {
	case statev1.InviteStatus_PENDING:
		return loc.Sprintf("label.invite_pending"), "warning"
	case statev1.InviteStatus_CLAIMED:
		return loc.Sprintf("label.invite_claimed"), "success"
	case statev1.InviteStatus_REVOKED:
		return loc.Sprintf("label.invite_revoked"), "error"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// formatCreatedDate returns a YYYY-MM-DD string for a timestamp.
func formatCreatedDate(createdAt *timestamppb.Timestamp) string {
	if createdAt == nil {
		return ""
	}
	return createdAt.AsTime().Format("2006-01-02")
}

// formatTimestamp returns a YYYY-MM-DD HH:MM:SS string for a timestamp.
func formatTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return value.AsTime().Format("2006-01-02 15:04:05")
}

// truncateText shortens text to a maximum length with an ellipsis.
func truncateText(text string, limit int) string {
	if limit <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}
