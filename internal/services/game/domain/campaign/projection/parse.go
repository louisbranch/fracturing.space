package projection

import (
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
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

func parseCampaignStatus(value string) (campaign.CampaignStatus, error) {
	return campaign.CampaignStatusFromLabel(value)
}

func parseGmMode(value string) (campaign.GmMode, error) {
	return campaign.GmModeFromLabel(value)
}

func parseCampaignIntent(value string) campaign.CampaignIntent {
	return campaign.CampaignIntentFromLabel(value)
}

func parseCampaignAccessPolicy(value string) campaign.CampaignAccessPolicy {
	return campaign.CampaignAccessPolicyFromLabel(value)
}

func parseParticipantRole(value string) (participant.ParticipantRole, error) {
	return participant.ParticipantRoleFromLabel(value)
}

func parseParticipantController(value string) (participant.Controller, error) {
	return participant.ControllerFromLabel(value)
}

func parseCampaignAccess(value string) (participant.CampaignAccess, error) {
	return participant.CampaignAccessFromLabel(value)
}

func parseInviteStatus(value string) (invite.Status, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invite.StatusUnspecified, fmt.Errorf("invite status is required")
	}
	status := invite.StatusFromLabel(trimmed)
	if status == invite.StatusUnspecified {
		return invite.StatusUnspecified, fmt.Errorf("unknown invite status: %s", trimmed)
	}
	return status, nil
}

func parseCharacterKind(value string) (character.CharacterKind, error) {
	return character.CharacterKindFromLabel(value)
}
