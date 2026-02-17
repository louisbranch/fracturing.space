package invite

// State captures replayed invite lifecycle state.
//
// The invite aggregate exists so a claim/revoke flow can be validated against
// a stable, replayable history instead of ephemeral request context.
type State struct {
	// Created indicates an invite record has been provisioned.
	Created bool
	// InviteID is the immutable invite token/id used in all command routing.
	InviteID string
	// ParticipantID identifies who can claim or manage this invite.
	ParticipantID string
	// RecipientUserID is the intended user or identity claim for this invite.
	RecipientUserID string
	// CreatedByParticipantID stores who issued the invite for auditability.
	CreatedByParticipantID string
	// Status is the current lifecycle phase of the invite.
	Status string
}
