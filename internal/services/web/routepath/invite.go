package routepath

const (
	InvitePrefix               = "/invite/"
	PublicInvitePattern        = InvitePrefix + "{inviteID}"
	PublicInviteRestPattern    = InvitePrefix + "{inviteID}/{rest...}"
	PublicInviteAcceptPattern  = InvitePrefix + "{inviteID}/accept"
	PublicInviteDeclinePattern = InvitePrefix + "{inviteID}/decline"
)

// PublicInvite returns the public invite route.
func PublicInvite(inviteID string) string {
	return InvitePrefix + escapeSegment(inviteID)
}

// PublicInviteAccept returns the public invite accept route.
func PublicInviteAccept(inviteID string) string {
	return PublicInvite(inviteID) + "/accept"
}

// PublicInviteDecline returns the public invite decline route.
func PublicInviteDecline(inviteID string) string {
	return PublicInvite(inviteID) + "/decline"
}
