package web

import (
	"context"
	"net/http"
	"strings"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PrincipalNotificationClient is the narrow notification surface needed by unread badge resolution.
type PrincipalNotificationClient interface {
	GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error)
}

// PrincipalSocialClient is the narrow social surface needed by viewer chrome resolution.
type PrincipalSocialClient interface {
	GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error)
}

// viewerResolver resolves authenticated viewer chrome (display name, avatar, profile link, unread notifications).
type viewerResolver struct {
	socialClient       PrincipalSocialClient
	notificationClient PrincipalNotificationClient
	assetBaseURL       string
	resolveUserID      func(*http.Request) string
}

func newViewerResolver(
	socialClient PrincipalSocialClient,
	notificationClient PrincipalNotificationClient,
	assetBaseURL string,
	resolveUserID func(*http.Request) string,
) viewerResolver {
	return viewerResolver{
		socialClient:       socialClient,
		notificationClient: notificationClient,
		assetBaseURL:       assetBaseURL,
		resolveUserID:      resolveUserID,
	}
}

func (r viewerResolver) resolveViewerUncached(request *http.Request) module.Viewer {
	userID := r.resolveUserID(request)
	if userID == "" {
		return module.Viewer{}
	}
	viewer := module.Viewer{
		DisplayName:            "Adventurer",
		AvatarURL:              websupport.AvatarImageURL(r.assetBaseURL, "user", userID, "", ""),
		ProfileURL:             routepath.AppSettingsProfile,
		HasUnreadNotifications: r.resolveHasUnreadNotifications(contextFromRequest(request), userID),
	}
	if r.socialClient == nil {
		return viewer
	}
	ctx := grpcauthctx.WithUserID(request.Context(), userID)
	resp, err := r.socialClient.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: userID})
	if err != nil || resp == nil || resp.GetUserProfile() == nil {
		if status.Code(err) == codes.NotFound {
			viewer.ProfileURL = routepath.AppSettingsProfileWithNotice(routepath.SettingsNoticePublicProfileRequired)
		}
		return viewer
	}
	record := resp.GetUserProfile()
	username := strings.TrimSpace(record.GetUsername())
	if username != "" {
		viewer.ProfileURL = routepath.UserProfile(username)
	} else {
		viewer.ProfileURL = routepath.AppSettingsProfileWithNotice(routepath.SettingsNoticePublicProfileRequired)
	}
	if name := strings.TrimSpace(record.GetName()); name != "" {
		viewer.DisplayName = name
	} else if username != "" {
		viewer.DisplayName = username
	}
	avatarSetID := strings.TrimSpace(record.GetAvatarSetId())
	avatarAssetID := strings.TrimSpace(record.GetAvatarAssetId())
	if avatarSetID != "" || avatarAssetID != "" {
		viewer.AvatarURL = websupport.AvatarImageURL(
			r.assetBaseURL,
			"user",
			userID,
			avatarSetID,
			avatarAssetID,
		)
	}
	return viewer
}

func (r viewerResolver) resolveHasUnreadNotifications(ctx context.Context, userID string) bool {
	if r.notificationClient == nil {
		return false
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}
	resp, err := r.notificationClient.GetUnreadNotificationStatus(
		grpcauthctx.WithUserID(ctx, userID),
		&notificationsv1.GetUnreadNotificationStatusRequest{},
	)
	if err != nil || resp == nil {
		return false
	}
	if resp.GetHasUnread() {
		return true
	}
	return resp.GetUnreadCount() > 0
}

func (r viewerResolver) resolveViewer(request *http.Request) module.Viewer {
	if state := requestPrincipalStateFromRequest(request); state != nil {
		state.viewerOnce.Do(func() {
			state.viewer = r.resolveViewerUncached(request)
		})
		return state.viewer
	}
	return r.resolveViewerUncached(request)
}
