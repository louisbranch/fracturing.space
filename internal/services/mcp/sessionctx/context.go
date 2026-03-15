package sessionctx

// Context is the fixed MCP authority bound to one bridge session.
type Context struct {
	CampaignID    string
	SessionID     string
	ParticipantID string
}
