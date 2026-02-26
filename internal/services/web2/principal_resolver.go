package web2

import (
	"context"
	"net/http"
	"strings"
	"sync"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/authctx"
	web2i18n "github.com/louisbranch/fracturing.space/internal/services/web2/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/sessioncookie"
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
	authClient        module.AuthClient
	accountClient     module.AccountClient
	connectionsClient connectionsv1.ConnectionsServiceClient
	assetBaseURL      string
}

func newPrincipalResolver(cfg Config) principalResolver {
	return principalResolver{
		authClient:        cfg.AuthClient,
		accountClient:     cfg.AccountClient,
		connectionsClient: cfg.ConnectionsClient,
		assetBaseURL:      cfg.AssetBaseURL,
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
		DisplayName: "Adventurer",
		AvatarURL:   websupport.AvatarImageURL(r.assetBaseURL, "user", userID, "", ""),
	}
	if r.connectionsClient == nil {
		return viewer
	}
	ctx := grpcauthctx.WithUserID(request.Context(), userID)
	resp, err := r.connectionsClient.GetUserProfile(ctx, &connectionsv1.GetUserProfileRequest{UserId: userID})
	if err != nil || resp == nil || resp.GetUserProfile() == nil {
		return viewer
	}
	record := resp.GetUserProfile()
	if name := strings.TrimSpace(record.GetName()); name != "" {
		viewer.DisplayName = name
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
	fallback := web2i18n.ResolveTag(request, nil).String()
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
