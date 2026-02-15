package projection

import (
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func parseGameSystem(value string) (commonv1.GameSystem, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("game system is required")
	}
	if system, ok := commonv1.GameSystem_value[trimmed]; ok {
		return commonv1.GameSystem(system), nil
	}
	upper := strings.ToUpper(trimmed)
	if system, ok := commonv1.GameSystem_value["GAME_SYSTEM_"+upper]; ok {
		return commonv1.GameSystem(system), nil
	}
	return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("unknown game system: %s", trimmed)
}

func parseCampaignStatus(value string) (campaign.Status, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return campaign.StatusUnspecified, fmt.Errorf("campaign status is required")
	}
	if normalized, ok := campaign.NormalizeStatus(trimmed); ok {
		return normalized, nil
	}
	return campaign.StatusUnspecified, fmt.Errorf("unknown campaign status: %s", trimmed)
}

func parseGmMode(value string) (campaign.GmMode, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return campaign.GmModeUnspecified, fmt.Errorf("gm mode is required")
	}
	if normalized, ok := campaign.NormalizeGmMode(trimmed); ok {
		return normalized, nil
	}
	return campaign.GmModeUnspecified, fmt.Errorf("unknown gm mode: %s", trimmed)
}

func parseCampaignIntent(value string) campaign.Intent {
	return campaign.NormalizeIntent(value)
}

func parseCampaignAccessPolicy(value string) campaign.AccessPolicy {
	return campaign.NormalizeAccessPolicy(value)
}

func parseParticipantRole(value string) (participant.Role, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return participant.RoleUnspecified, fmt.Errorf("participant role is required")
	}
	if normalized, ok := participant.NormalizeRole(trimmed); ok {
		return normalized, nil
	}
	return participant.RoleUnspecified, fmt.Errorf("unknown participant role: %s", trimmed)
}

func parseParticipantController(value string) (participant.Controller, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return participant.ControllerUnspecified, fmt.Errorf("participant controller is required")
	}
	if normalized, ok := participant.NormalizeController(trimmed); ok {
		return normalized, nil
	}
	return participant.ControllerUnspecified, fmt.Errorf("unknown participant controller: %s", trimmed)
}

func parseCampaignAccess(value string) (participant.CampaignAccess, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return participant.CampaignAccessUnspecified, fmt.Errorf("campaign access is required")
	}
	if normalized, ok := participant.NormalizeCampaignAccess(trimmed); ok {
		return normalized, nil
	}
	return participant.CampaignAccessUnspecified, fmt.Errorf("unknown campaign access: %s", trimmed)
}

func parseInviteStatus(value string) (invite.Status, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invite.StatusUnspecified, fmt.Errorf("invite status is required")
	}
	if normalized, ok := invite.NormalizeStatus(trimmed); ok {
		return normalized, nil
	}
	return invite.StatusUnspecified, fmt.Errorf("unknown invite status: %s", trimmed)
}

func parseCharacterKind(value string) (character.Kind, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return character.KindUnspecified, fmt.Errorf("character kind is required")
	}
	if normalized, ok := character.NormalizeKind(trimmed); ok {
		return normalized, nil
	}
	return character.KindUnspecified, fmt.Errorf("unknown character kind: %s", trimmed)
}
