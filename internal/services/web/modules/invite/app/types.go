package app

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
)

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

// Gateway loads and mutates public invite workflows.
type Gateway interface {
	GetPublicInvite(context.Context, string) (PublicInvite, error)
	AcceptInvite(context.Context, string, PublicInvite) error
	DeclineInvite(context.Context, string, string) error
}

// Service exposes invite landing workflows used by transport handlers.
type Service interface {
	LoadInvite(context.Context, string, string) (InvitePage, error)
	AcceptInvite(context.Context, string, string) (InviteMutationResult, error)
	DeclineInvite(context.Context, string, string) (InviteMutationResult, error)
}

// RequireUserID validates and returns a normalized viewer user ID.
func RequireUserID(userID string) (string, error) {
	return userid.Require(userID)
}
