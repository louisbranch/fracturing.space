package web

import (
	"context"
	"net/http"
	"strings"
	"sync"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/authctx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type requestPrincipalState struct {
	userIDOnce   sync.Once
	userID       string
	viewerOnce   sync.Once
	viewer       module.Viewer
	languageOnce sync.Once
	language     string
}

type requestPrincipalStateKey struct{}

type principalResolver struct {
	authClient         module.AuthClient
	accountClient      module.AccountClient
	notificationClient module.NotificationClient
	socialClient       socialv1.SocialServiceClient
	assetBaseURL       string
}

func newPrincipalResolver(cfg Config) principalResolver {
	return principalResolver{
		authClient:         cfg.AuthClient,
		accountClient:      cfg.AccountClient,
		notificationClient: cfg.NotificationClient,
		socialClient:       cfg.SocialClient,
		assetBaseURL:       cfg.AssetBaseURL,
	}
}

func (r principalResolver) resolveSessionUserID(ctx context.Context, sessionID string) (string, bool) {
	if r.authClient == nil {
		return "", false
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", false
	}
	resp, err := r.authClient.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil || resp == nil || resp.GetSession() == nil {
		return "", false
	}
	userID := strings.TrimSpace(resp.GetSession().GetUserId())
	if userID == "" {
		return "", false
	}
	return userID, true
}

func (r principalResolver) resolveRequestUserIDUncached(req *http.Request) string {
	if req == nil {
		return ""
	}
	sessionID, ok := sessioncookie.Read(req)
	if !ok {
		return ""
	}
	userID, ok := r.resolveSessionUserID(req.Context(), sessionID)
	if !ok {
		return ""
	}
	return userID
}

func (r principalResolver) resolveRequestUserID(request *http.Request) string {
	if state := requestPrincipalStateFromRequest(request); state != nil {
		state.userIDOnce.Do(func() {
			state.userID = r.resolveRequestUserIDUncached(request)
		})
		return state.userID
	}
	return r.resolveRequestUserIDUncached(request)
}

func (r principalResolver) resolveViewerUncached(request *http.Request) module.Viewer {
	userID := r.resolveRequestUserID(request)
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

func (r principalResolver) resolveHasUnreadNotifications(ctx context.Context, userID string) bool {
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

func contextFromRequest(request *http.Request) context.Context {
	if request == nil {
		return context.Background()
	}
	return request.Context()
}

func (r principalResolver) resolveViewer(request *http.Request) module.Viewer {
	if state := requestPrincipalStateFromRequest(request); state != nil {
		state.viewerOnce.Do(func() {
			state.viewer = r.resolveViewerUncached(request)
		})
		return state.viewer
	}
	return r.resolveViewerUncached(request)
}

func (r principalResolver) resolveRequestLanguageUncached(request *http.Request) string {
	fallback := webi18n.ResolveTag(request, nil).String()
	if r.accountClient == nil {
		return fallback
	}
	userID := r.resolveRequestUserID(request)
	if userID == "" {
		return fallback
	}
	resp, err := r.accountClient.GetProfile(request.Context(), &authv1.GetProfileRequest{UserId: userID})
	if err != nil || resp == nil || resp.GetProfile() == nil {
		return fallback
	}
	locale := resp.GetProfile().GetLocale()
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		return fallback
	}
	return platformi18n.LocaleString(platformi18n.NormalizeLocale(locale))
}

func (r principalResolver) resolveRequestLanguage(request *http.Request) string {
	if state := requestPrincipalStateFromRequest(request); state != nil {
		state.languageOnce.Do(func() {
			state.language = r.resolveRequestLanguageUncached(request)
		})
		return state.language
	}
	return r.resolveRequestLanguageUncached(request)
}

func (r principalResolver) authRequired() func(*http.Request) bool {
	validated := authctx.ValidatedSessionAuth(func(ctx context.Context, sessionID string) bool {
		userID, ok := r.resolveSessionUserID(ctx, sessionID)
		if !ok {
			return false
		}
		if state := requestPrincipalStateFromContext(ctx); state != nil {
			state.userIDOnce.Do(func() {
				state.userID = userID
			})
		}
		return true
	})
	return func(request *http.Request) bool {
		return validated(request)
	}
}
