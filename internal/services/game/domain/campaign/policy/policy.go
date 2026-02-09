// Package policy provides authorization decisions for state actions.
package policy

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
)

// Action represents a policy decision for a participant action.
type Action int

const (
	// ActionManageParticipants allows managing participants.
	ActionManageParticipants Action = iota + 1
	// ActionManageInvites allows managing invites.
	ActionManageInvites
)

// Can reports whether the participant can perform the action for the campaign.
//
// v0 policy: managers and owners can manage participants and invites; others cannot.
func Can(actor participant.Participant, action Action, _ campaign.Campaign) bool {
	if action != ActionManageParticipants && action != ActionManageInvites {
		return false
	}
	return actor.CampaignAccess == participant.CampaignAccessOwner || actor.CampaignAccess == participant.CampaignAccessManager
}
