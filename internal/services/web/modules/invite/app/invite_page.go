package app

// InviteStatus captures the public invite lifecycle states visible in web.
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusClaimed  InviteStatus = "claimed"
	InviteStatusDeclined InviteStatus = "declined"
	InviteStatusRevoked  InviteStatus = "revoked"
)

// InvitePageState captures the viewer-specific landing experience for one invite.
type InvitePageState string

const (
	InvitePageStateAnonymous InvitePageState = "anonymous"
	InvitePageStateClaimable InvitePageState = "claimable"
	InvitePageStateTargeted  InvitePageState = "targeted"
	InvitePageStateMismatch  InvitePageState = "mismatch"
	InvitePageStateClaimed   InvitePageState = "claimed"
	InvitePageStateDeclined  InvitePageState = "declined"
	InvitePageStateRevoked   InvitePageState = "revoked"
)

// PublicInvite stores the public landing information for one invite.
type PublicInvite struct {
	InviteID        string
	CampaignID      string
	CampaignName    string
	CampaignStatus  string
	ParticipantID   string
	ParticipantName string
	RecipientUserID string
	CreatedByUserID string
	InviterUsername string
	Status          InviteStatus
}

// InvitePage stores invite data plus viewer-specific state for rendering.
type InvitePage struct {
	Invite       PublicInvite
	ViewerUserID string
	State        InvitePageState
	CanAccept    bool
	CanDecline   bool
}

// InviteMutationResult identifies affected dashboard scopes after a mutation.
type InviteMutationResult struct {
	CampaignID string
	UserIDs    []string
}
