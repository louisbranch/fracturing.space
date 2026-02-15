package invite

// State captures invite facts derived from domain events.
type State struct {
	Created                bool
	InviteID               string
	ParticipantID          string
	RecipientUserID        string
	CreatedByParticipantID string
	Status                 string
}
