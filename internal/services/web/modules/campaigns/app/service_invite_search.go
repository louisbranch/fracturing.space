package app

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
)

// SearchInviteUsers centralizes invite typeahead search for one campaign.
func (s inviteReadService) SearchInviteUsers(ctx context.Context, campaignID string, input SearchInviteUsersInput) ([]InviteUserSearchResult, error) {
	return s.searchInviteUsers(ctx, campaignID, input)
}

// searchInviteUsers loads ranked invite-search matches for one campaign manager.
func (s inviteReadService) searchInviteUsers(ctx context.Context, campaignID string, input SearchInviteUsersInput) ([]InviteUserSearchResult, error) {
	if err := s.auth.requirePolicy(ctx, campaignID, policyManageInvite); err != nil {
		return nil, err
	}
	viewerUserID, err := userid.Require(input.ViewerUserID)
	if err != nil {
		return nil, err
	}
	input.ViewerUserID = viewerUserID
	input.Query = strings.TrimSpace(input.Query)
	if input.Limit <= 0 {
		input.Limit = 8
	}
	return s.read.SearchInviteUsers(ctx, input)
}
