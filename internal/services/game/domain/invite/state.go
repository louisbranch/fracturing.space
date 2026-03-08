package invite

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// State captures replayed invite lifecycle state.
//
// The invite aggregate exists so a claim/revoke flow can be validated against
// a stable, replayable history instead of ephemeral request context.
type State struct {
	// Created indicates an invite record has been provisioned.
	Created bool
	// InviteID is the immutable invite token/id used in all command routing.
	InviteID ids.InviteID
	// ParticipantID identifies who can claim or manage this invite.
	ParticipantID ids.ParticipantID
	// RecipientUserID is the intended user or identity claim for this invite.
	RecipientUserID ids.UserID
	// CreatedByParticipantID stores who issued the invite for auditability.
	CreatedByParticipantID ids.ParticipantID
	// Status is the current lifecycle phase of the invite.
	Status string
}
