package participant

// State captures participant facts derived from domain events.
type State struct {
	Joined         bool
	Left           bool
	ParticipantID  string
	UserID         string
	DisplayName    string
	Role           string
	Controller     string
	CampaignAccess string
}
