package game

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"

func shouldClearCampaignAIBindingOnAccessChange(before participant.CampaignAccess, after participant.CampaignAccess) bool {
	if before == after {
		return false
	}
	return before == participant.CampaignAccessOwner || after == participant.CampaignAccessOwner
}
