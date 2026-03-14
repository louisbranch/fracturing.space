package app

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
)

const minInviteSearchQueryLength = 2

// searchInviteUsers loads ranked invite-search matches for one campaign manager.
func (s service) searchInviteUsers(ctx context.Context, campaignID string, input SearchInviteUsersInput) ([]InviteUserSearchResult, error) {
	if err := s.requirePolicy(ctx, campaignID, policyManageInvite); err != nil {
		return nil, err
	}
	viewerUserID, err := userid.Require(input.ViewerUserID)
	if err != nil {
		return nil, err
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	if len(query) < minInviteSearchQueryLength {
		return []InviteUserSearchResult{}, nil
	}
	input.ViewerUserID = viewerUserID
	input.Query = query
	if input.Limit <= 0 {
		input.Limit = 8
	}
	return s.readGateway.SearchInviteUsers(ctx, input)
}
