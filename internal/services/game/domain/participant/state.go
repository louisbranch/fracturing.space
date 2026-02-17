package participant

// State captures replayed campaign membership and control intent.
//
// Permission checks in multiple services derive from this snapshot to avoid
// scattering identity and role logic across handlers.
type State struct {
	// Joined indicates the participant currently exists in this campaign roster.
	Joined bool
	// Left indicates a completed leave command has been processed.
	Left bool
	// ParticipantID is the campaign-scoped identity used by domain commands.
	ParticipantID string
	// UserID links participant records to external authentication identities.
	UserID string
	// Name is shown across campaign/session UI and projection outputs.
	Name string
	// Role is the campaign role used for authorization decisions.
	Role string
	// Controller indicates who can command actions for this participant.
	Controller string
	// CampaignAccess controls visibility and permission scope at campaign level.
	CampaignAccess string
}
