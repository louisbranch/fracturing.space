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
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
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

// newViewerResolver builds package wiring for this web seam.
func newViewerResolver(
	socialClient PrincipalSocialClient,
	notificationClient PrincipalNotificationClient,
	assetBaseURL string,
	resolveUserID func(*http.Request) string,
) viewerResolver {
	if resolveUserID == nil {
		resolveUserID = func(*http.Request) string { return "" }
	}
	return viewerResolver{
		socialClient:       socialClient,
		notificationClient: notificationClient,
		assetBaseURL:       assetBaseURL,
		resolveUserID:      resolveUserID,
	}
}

// resolveViewerUncached resolves request-scoped values needed by this package.
func (r viewerResolver) resolveViewerUncached(request *http.Request) module.Viewer {
	if request == nil {
		return module.Viewer{}
	}
	userID := userid.Normalize(r.resolveUserID(request))
	if userID == "" {
		return module.Viewer{}
	}
	viewer := defaultViewer(r.assetBaseURL, contextFromRequest(request), userID, r.resolveHasUnreadNotifications)
	if r.socialClient == nil {
		return viewer
	}
	record, err := r.loadUserProfile(request.Context(), userID)
	return applyUserProfile(viewer, r.assetBaseURL, userID, record, err)
}

// resolveHasUnreadNotifications resolves request-scoped values needed by this package.
func (r viewerResolver) resolveHasUnreadNotifications(ctx context.Context, userID string) bool {
	if r.notificationClient == nil {
		return false
	}
	userID = userid.Normalize(userID)
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

// resolveViewer resolves request-scoped values needed by this package.
func (r viewerResolver) resolveViewer(request *http.Request) module.Viewer {
	if state := requestPrincipalStateFromRequest(request); state != nil {
		state.viewerOnce.Do(func() {
			state.viewer = r.resolveViewerUncached(request)
		})
		return state.viewer
	}
	return r.resolveViewerUncached(request)
}

// defaultViewer returns the fallback viewer state used before optional social-profile enrichment.
func defaultViewer(
	assetBaseURL string,
	ctx context.Context,
	userID string,
	resolveUnread func(context.Context, string) bool,
) module.Viewer {
	viewer := module.Viewer{
		DisplayName: "Adventurer",
		AvatarURL:   websupport.AvatarImageURL(assetBaseURL, "user", userID, "", ""),
		ProfileURL:  routepath.AppSettingsProfile,
	}
	if resolveUnread != nil {
		viewer.HasUnreadNotifications = resolveUnread(ctx, userID)
	}
	return viewer
}

// loadUserProfile fetches the profile record used to personalize viewer chrome.
func (r viewerResolver) loadUserProfile(ctx context.Context, userID string) (*socialv1.UserProfile, error) {
	ctx = grpcauthctx.WithUserID(ctx, userID)
	resp, err := r.socialClient.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: userID})
	if err != nil || resp == nil {
		return nil, err
	}
	return resp.GetUserProfile(), nil
}

// applyUserProfile merges social profile details into default viewer chrome.
func applyUserProfile(
	viewer module.Viewer,
	assetBaseURL string,
	userID string,
	record *socialv1.UserProfile,
	err error,
) module.Viewer {
	if record == nil {
		if status.Code(err) == codes.NotFound {
			viewer.ProfileURL = routepath.AppSettingsProfileRequired
		}
		return viewer
	}

	username := strings.TrimSpace(record.GetUsername())
	if username != "" {
		viewer.ProfileURL = routepath.UserProfile(username)
	} else {
		viewer.ProfileURL = routepath.AppSettingsProfileRequired
	}

	name := strings.TrimSpace(record.GetName())
	if name != "" {
		viewer.DisplayName = name
	} else if username != "" {
		viewer.DisplayName = username
	}

	avatarSetID := strings.TrimSpace(record.GetAvatarSetId())
	avatarAssetID := strings.TrimSpace(record.GetAvatarAssetId())
	if avatarSetID != "" || avatarAssetID != "" {
		viewer.AvatarURL = websupport.AvatarImageURL(assetBaseURL, "user", userID, avatarSetID, avatarAssetID)
	}
	return viewer
}
