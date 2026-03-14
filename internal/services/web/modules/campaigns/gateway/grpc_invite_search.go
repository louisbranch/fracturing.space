package gateway

import (
	"context"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// SearchInviteUsers loads ranked invite-search matches from social.
func (g inviteReadGateway) SearchInviteUsers(ctx context.Context, input campaignapp.SearchInviteUsersInput) ([]campaignapp.InviteUserSearchResult, error) {
	if g.read.Social == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.social_service_client_is_not_configured", "social service client is not configured")
	}
	resp, err := g.read.Social.SearchUsers(ctx, &socialv1.SearchUsersRequest{
		ViewerUserId: strings.TrimSpace(input.ViewerUserID),
		Query:        strings.TrimSpace(input.Query),
		Limit:        int32(input.Limit),
	})
	if err != nil {
		return nil, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackKey:     "error.web.message.failed_to_search_users",
			FallbackMessage: "failed to search users",
		})
	}
	results := make([]campaignapp.InviteUserSearchResult, 0, len(resp.GetUsers()))
	for _, user := range resp.GetUsers() {
		if user == nil {
			continue
		}
		results = append(results, campaignapp.InviteUserSearchResult{
			UserID:        strings.TrimSpace(user.GetUserId()),
			Username:      strings.TrimSpace(user.GetUsername()),
			Name:          strings.TrimSpace(user.GetName()),
			AvatarSetID:   strings.TrimSpace(user.GetAvatarSetId()),
			AvatarAssetID: strings.TrimSpace(user.GetAvatarAssetId()),
			IsContact:     user.GetIsContact(),
		})
	}
	return results, nil
}
