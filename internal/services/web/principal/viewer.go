package principal

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
)

// ResolveViewer returns request-scoped viewer chrome for authenticated pages.
func (r Resolver) ResolveViewer(request *http.Request) module.Viewer {
	if snapshot := snapshotFromRequest(request); snapshot != nil {
		snapshot.viewerOnce.Do(func() {
			snapshot.viewer = r.resolveViewerUncached(request)
		})
		return snapshot.viewer
	}
	return r.resolveViewerUncached(request)
}

// ResolveRequestViewer adapts the production resolver to the shared page
// contract used by transport helpers.
func (r Resolver) ResolveRequestViewer(request *http.Request) module.Viewer {
	return r.ResolveViewer(request)
}

// resolveViewerUncached resolves viewer chrome without using the request
// snapshot, which is primarily useful for plain unit tests.
func (r Resolver) resolveViewerUncached(request *http.Request) module.Viewer {
	if request == nil {
		return module.Viewer{}
	}
	userID := userid.Normalize(r.ResolveUserID(request))
	if userID == "" {
		return module.Viewer{}
	}
	viewer := defaultViewer(r.viewer.assetBaseURL, contextFromRequest(request), userID, r.resolveHasUnreadNotifications)
	viewer.NotificationsAvailable = r.viewer.notification != nil
	if profile := r.loadAccountProfile(request.Context(), userID); profile != nil {
		username := strings.TrimSpace(profile.GetUsername())
		if username != "" {
			viewer.ProfileURL = routepath.UserProfile(username)
		}
	}
	if r.viewer.social == nil {
		return viewer
	}
	return applyUserProfile(viewer, r.viewer.assetBaseURL, userID, r.loadUserProfile(request.Context(), userID))
}

// resolveHasUnreadNotifications loads the request user's unread-badge state.
func (r Resolver) resolveHasUnreadNotifications(ctx context.Context, userID string) bool {
	if r.viewer.notification == nil {
		return false
	}
	userID = userid.Normalize(userID)
	if userID == "" {
		return false
	}
	resp, err := r.viewer.notification.GetUnreadNotificationStatus(
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

// loadUserProfile fetches optional social profile data for viewer chrome
// personalization.
func (r Resolver) loadUserProfile(ctx context.Context, userID string) *socialv1.UserProfile {
	if r.viewer.social == nil {
		return nil
	}
	resp, err := r.viewer.social.GetUserProfile(
		grpcauthctx.WithUserID(ctx, userID),
		&socialv1.GetUserProfileRequest{UserId: userID},
	)
	if err != nil || resp == nil {
		return nil
	}
	return resp.GetUserProfile()
}

// defaultViewer defines the authenticated chrome fallback before optional
// social-profile enrichment.
func defaultViewer(
	assetBaseURL string,
	ctx context.Context,
	userID string,
	resolveUnread func(context.Context, string) bool,
) module.Viewer {
	viewer := module.Viewer{
		DisplayName: "Adventurer",
		AvatarURL:   websupport.AvatarImageURL(assetBaseURL, "user", userID, "", "", viewerAvatarDeliveryWidthPX),
		ProfileURL:  routepath.AppDashboard,
	}
	if resolveUnread != nil {
		viewer.HasUnreadNotifications = resolveUnread(ctx, userID)
	}
	return viewer
}

// applyUserProfile merges social profile data into the default viewer chrome.
func applyUserProfile(
	viewer module.Viewer,
	assetBaseURL string,
	userID string,
	record *socialv1.UserProfile,
) module.Viewer {
	if record == nil {
		return viewer
	}

	name := strings.TrimSpace(record.GetName())
	if name != "" {
		viewer.DisplayName = name
	}

	avatarSetID := strings.TrimSpace(record.GetAvatarSetId())
	avatarAssetID := strings.TrimSpace(record.GetAvatarAssetId())
	if avatarSetID != "" || avatarAssetID != "" {
		viewer.AvatarURL = websupport.AvatarImageURL(assetBaseURL, "user", userID, avatarSetID, avatarAssetID, viewerAvatarDeliveryWidthPX)
	}
	return viewer
}
