package principal

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
	sharedhttpx "github.com/louisbranch/fracturing.space/internal/services/shared/httpx"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/authctx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc"
)

// SessionClient is the narrow auth surface needed for session validation.
type SessionClient interface {
	GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error)
}

// AccountClient is the narrow auth account surface needed for locale and
// profile-link resolution.
type AccountClient interface {
	GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error)
}

// NotificationClient is the narrow notifications surface needed for unread
// badge resolution.
type NotificationClient interface {
	GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error)
}

// SocialClient is the narrow social surface needed for viewer personalization.
type SocialClient interface {
	GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error)
}

// Dependencies carries the clients required for request-scoped principal
// resolution. These clients intentionally mirror browser concerns rather than
// feature-module ownership.
type Dependencies struct {
	SessionClient      SessionClient
	AccountClient      AccountClient
	NotificationClient NotificationClient
	SocialClient       SocialClient
	AssetBaseURL       string
}

// Resolver centralizes request-scoped session, viewer, and language
// resolution behind one explicit package seam.
type Resolver struct {
	deps Dependencies
}

// requestSnapshot caches principal lookups for one request so repeated handler
// and template resolution does not duplicate backend calls.
type requestSnapshot struct {
	userIDOnce         sync.Once
	userID             string
	viewerOnce         sync.Once
	viewer             module.Viewer
	languageOnce       sync.Once
	language           string
	accountProfileOnce sync.Once
	accountProfile     *authv1.AccountProfile
}

// snapshotContextKey keeps the request snapshot private to this package.
type snapshotContextKey struct{}

const viewerAvatarDeliveryWidthPX = 40

// New builds a resolver from startup dependencies.
func New(deps Dependencies) Resolver {
	return Resolver{deps: deps}
}

// Middleware seeds the request-scoped principal snapshot used by the resolver.
func (r Resolver) Middleware() sharedhttpx.Middleware {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if request == nil {
				next.ServeHTTP(w, request)
				return
			}
			snapshot := &requestSnapshot{}
			ctx := context.WithValue(request.Context(), snapshotContextKey{}, snapshot)
			next.ServeHTTP(w, request.WithContext(ctx))
		})
	}
}

// AuthRequired returns the auth gate used by the protected app surface.
func (r Resolver) AuthRequired() func(*http.Request) bool {
	validated := authctx.ValidatedSessionAuth(func(ctx context.Context, sessionID string) bool {
		userID, ok := r.resolveSessionUserID(ctx, sessionID)
		if !ok {
			return false
		}
		if snapshot := snapshotFromContext(ctx); snapshot != nil {
			snapshot.userIDOnce.Do(func() {
				snapshot.userID = userID
			})
		}
		return true
	})
	return func(request *http.Request) bool {
		return validated(request)
	}
}

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

// ResolveSignedIn reports whether the request carries a valid authenticated
// user.
func (r Resolver) ResolveSignedIn(request *http.Request) bool {
	return userid.Normalize(r.ResolveUserID(request)) != ""
}

// ResolveUserID returns the authenticated user id for the request when one can
// be resolved safely.
func (r Resolver) ResolveUserID(request *http.Request) string {
	if snapshot := snapshotFromRequest(request); snapshot != nil {
		snapshot.userIDOnce.Do(func() {
			snapshot.userID = r.resolveUserIDUncached(request)
		})
		return snapshot.userID
	}
	return r.resolveUserIDUncached(request)
}

// ResolveLanguage returns the effective request language, preferring the
// account locale for authenticated requests.
func (r Resolver) ResolveLanguage(request *http.Request) string {
	if snapshot := snapshotFromRequest(request); snapshot != nil {
		snapshot.languageOnce.Do(func() {
			snapshot.language = r.resolveLanguageUncached(request)
		})
		return snapshot.language
	}
	return r.resolveLanguageUncached(request)
}

// resolveSessionUserID validates the session cookie value and normalizes the
// resulting user id for downstream browser use.
func (r Resolver) resolveSessionUserID(ctx context.Context, sessionID string) (string, bool) {
	if r.deps.SessionClient == nil {
		return "", false
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", false
	}
	resp, err := r.deps.SessionClient.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil || resp == nil || resp.GetSession() == nil {
		return "", false
	}
	userID := userid.Normalize(resp.GetSession().GetUserId())
	if userID == "" {
		return "", false
	}
	return userID, true
}

// resolveUserIDUncached reads and validates the request session cookie.
func (r Resolver) resolveUserIDUncached(request *http.Request) string {
	if request == nil {
		return ""
	}
	sessionID, ok := sessioncookie.Read(request)
	if !ok {
		return ""
	}
	userID, ok := r.resolveSessionUserID(request.Context(), sessionID)
	if !ok {
		return ""
	}
	return userID
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
	viewer := defaultViewer(r.deps.AssetBaseURL, contextFromRequest(request), userID, r.resolveHasUnreadNotifications)
	viewer.NotificationsAvailable = r.deps.NotificationClient != nil
	if profile := r.loadAccountProfile(request.Context(), userID); profile != nil {
		username := strings.TrimSpace(profile.GetUsername())
		if username != "" {
			viewer.ProfileURL = routepath.UserProfile(username)
		}
	}
	if r.deps.SocialClient == nil {
		return viewer
	}
	return applyUserProfile(viewer, r.deps.AssetBaseURL, userID, r.loadUserProfile(request.Context(), userID))
}

// resolveLanguageUncached prefers the authenticated account locale before
// falling back to transport language negotiation.
func (r Resolver) resolveLanguageUncached(request *http.Request) string {
	fallback := webi18n.ResolveTag(request, nil).String()
	if request == nil || r.deps.AccountClient == nil {
		return fallback
	}
	userID := userid.Normalize(r.ResolveUserID(request))
	if userID == "" {
		return fallback
	}
	profile := r.loadAccountProfile(request.Context(), userID)
	if profile == nil {
		return fallback
	}
	locale := profile.GetLocale()
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		return fallback
	}
	return platformi18n.LocaleString(platformi18n.NormalizeLocale(locale))
}

// resolveHasUnreadNotifications loads the request user's unread-badge state.
func (r Resolver) resolveHasUnreadNotifications(ctx context.Context, userID string) bool {
	if r.deps.NotificationClient == nil {
		return false
	}
	userID = userid.Normalize(userID)
	if userID == "" {
		return false
	}
	resp, err := r.deps.NotificationClient.GetUnreadNotificationStatus(
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

// loadAccountProfile returns auth-owned profile data and memoizes it inside the
// request snapshot so viewer and language resolution share one lookup.
func (r Resolver) loadAccountProfile(ctx context.Context, userID string) *authv1.AccountProfile {
	if r.deps.AccountClient == nil {
		return nil
	}
	if snapshot := snapshotFromContext(ctx); snapshot != nil {
		snapshot.accountProfileOnce.Do(func() {
			snapshot.accountProfile = r.loadAccountProfileUncached(ctx, userID)
		})
		return snapshot.accountProfile
	}
	return r.loadAccountProfileUncached(ctx, userID)
}

// loadAccountProfileUncached fetches auth-owned profile data without using the
// request snapshot.
func (r Resolver) loadAccountProfileUncached(ctx context.Context, userID string) *authv1.AccountProfile {
	resp, err := r.deps.AccountClient.GetProfile(ctx, &authv1.GetProfileRequest{UserId: userID})
	if err != nil || resp == nil {
		return nil
	}
	return resp.GetProfile()
}

// loadUserProfile fetches optional social profile data for viewer chrome
// personalization.
func (r Resolver) loadUserProfile(ctx context.Context, userID string) *socialv1.UserProfile {
	resp, err := r.deps.SocialClient.GetUserProfile(
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

// contextFromRequest keeps nil-request behavior consistent across principal
// resolution.
func contextFromRequest(request *http.Request) context.Context {
	if request == nil {
		return context.Background()
	}
	return request.Context()
}

// snapshotFromRequest returns the per-request principal snapshot when the
// middleware has attached one.
func snapshotFromRequest(request *http.Request) *requestSnapshot {
	if request == nil {
		return nil
	}
	return snapshotFromContext(request.Context())
}

// snapshotFromContext returns the per-request principal snapshot when present.
func snapshotFromContext(ctx context.Context) *requestSnapshot {
	if ctx == nil {
		return nil
	}
	snapshot, _ := ctx.Value(snapshotContextKey{}).(*requestSnapshot)
	return snapshot
}
