package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

// shouldClearCampaignAIBindingOnAccessChange protects the campaign owner-bound
// AI agent contract when participant ownership changes.
func shouldClearCampaignAIBindingOnAccessChange(before participant.CampaignAccess, after participant.CampaignAccess) bool {
	if before == after {
		return false
	}
	return before == participant.CampaignAccessOwner || after == participant.CampaignAccessOwner
}

// disallowsHumanGMForCampaignGMMode centralizes the campaign-level invariant
// that AI-gm campaigns only accept AI-controlled GM seats.
func disallowsHumanGMForCampaignGMMode(
	gmMode campaign.GmMode,
	role participant.Role,
	controller participant.Controller,
) bool {
	return gmMode == campaign.GmModeAI && role == participant.RoleGM && controller == participant.ControllerHuman
}
