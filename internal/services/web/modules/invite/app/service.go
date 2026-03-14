package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// service owns the invite landing decision logic behind the transport layer.
type service struct {
	gateway Gateway
}

// NewService constructs the public invite service with fail-closed defaults.
func NewService(gateway Gateway) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}

// LoadInvite resolves public invite data and viewer-specific action state.
func (s service) LoadInvite(ctx context.Context, viewerUserID string, inviteID string) (InvitePage, error) {
	inviteID = strings.TrimSpace(inviteID)
	if inviteID == "" {
		return InvitePage{}, apperrors.E(apperrors.KindNotFound, "invite not found")
	}
	invite, err := s.gateway.GetPublicInvite(ctx, inviteID)
	if err != nil {
		return InvitePage{}, err
	}
	viewerUserID = useridOrBlank(viewerUserID)
	page := InvitePage{
		Invite:       invite,
		ViewerUserID: viewerUserID,
	}

	switch invite.Status {
	case InviteStatusClaimed:
		page.State = InvitePageStateClaimed
		return page, nil
	case InviteStatusDeclined:
		page.State = InvitePageStateDeclined
		return page, nil
	case InviteStatusRevoked:
		page.State = InvitePageStateRevoked
		return page, nil
	}

	if viewerUserID == "" {
		page.State = InvitePageStateAnonymous
		return page, nil
	}
	if invite.RecipientUserID != "" && !strings.EqualFold(invite.RecipientUserID, viewerUserID) {
		page.State = InvitePageStateMismatch
		return page, nil
	}
	if invite.RecipientUserID != "" {
		page.State = InvitePageStateTargeted
		page.CanAccept = true
		page.CanDecline = true
		return page, nil
	}
	page.State = InvitePageStateClaimable
	page.CanAccept = true
	return page, nil
}

// AcceptInvite claims one invite for the signed-in viewer.
func (s service) AcceptInvite(ctx context.Context, viewerUserID string, inviteID string) (InviteMutationResult, error) {
	viewerUserID, err := RequireUserID(viewerUserID)
	if err != nil {
		return InviteMutationResult{}, err
	}
	page, err := s.LoadInvite(ctx, viewerUserID, inviteID)
	if err != nil {
		return InviteMutationResult{}, err
	}
	if !page.CanAccept {
		return InviteMutationResult{}, apperrors.E(apperrors.KindForbidden, "invite cannot be accepted")
	}
	if err := s.gateway.AcceptInvite(ctx, viewerUserID, page.Invite); err != nil {
		return InviteMutationResult{}, err
	}
	return InviteMutationResult{
		CampaignID: page.Invite.CampaignID,
		UserIDs:    inviteMutationUsers(page.Invite, viewerUserID),
	}, nil
}

// DeclineInvite declines one targeted invite for the signed-in viewer.
func (s service) DeclineInvite(ctx context.Context, viewerUserID string, inviteID string) (InviteMutationResult, error) {
	viewerUserID, err := RequireUserID(viewerUserID)
	if err != nil {
		return InviteMutationResult{}, err
	}
	page, err := s.LoadInvite(ctx, viewerUserID, inviteID)
	if err != nil {
		return InviteMutationResult{}, err
	}
	if !page.CanDecline {
		return InviteMutationResult{}, apperrors.E(apperrors.KindForbidden, "invite cannot be declined")
	}
	if err := s.gateway.DeclineInvite(ctx, viewerUserID, page.Invite.InviteID); err != nil {
		return InviteMutationResult{}, err
	}
	return InviteMutationResult{
		CampaignID: page.Invite.CampaignID,
		UserIDs:    inviteMutationUsers(page.Invite, viewerUserID),
	}, nil
}

// useridOrBlank normalizes optional viewer identity without enforcing auth.
func useridOrBlank(userID string) string {
	return strings.TrimSpace(userID)
}

// inviteMutationUsers computes the dashboard invalidation audience for invite
// mutations so web handlers can refresh both recipient and creator views.
func inviteMutationUsers(invite PublicInvite, viewerUserID string) []string {
	result := make([]string, 0, 2)
	if viewerUserID = strings.TrimSpace(viewerUserID); viewerUserID != "" {
		result = append(result, viewerUserID)
	}
	if creatorUserID := strings.TrimSpace(invite.CreatedByUserID); creatorUserID != "" && !strings.EqualFold(creatorUserID, viewerUserID) {
		result = append(result, creatorUserID)
	}
	return result
}
