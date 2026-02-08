// Package policy provides authorization decisions for state actions.
package policy

import (
	"github.com/louisbranch/fracturing.space/internal/campaign"
	"github.com/louisbranch/fracturing.space/internal/campaign/participant"
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
// v0 policy: owners can manage participants and invites; others cannot.
func Can(actor participant.Participant, action Action, _ campaign.Campaign) bool {
	if !actor.IsOwner {
		return false
	}
	return action == ActionManageParticipants || action == ActionManageInvites
}
