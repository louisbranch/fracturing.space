package principal

import (
	"context"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
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

// authResolver owns session validation and user-id resolution.
type authResolver struct {
	sessionClient SessionClient
}

// accountProfileResolver owns auth-backed profile loading shared by language
// and viewer resolution.
type accountProfileResolver struct {
	accountClient AccountClient
}

// viewerResolver owns authenticated viewer chrome assembly.
type viewerResolver struct {
	accountProfile accountProfileResolver
	notification   NotificationClient
	social         SocialClient
	assetBaseURL   string
}

// languageResolver owns locale resolution for authenticated requests.
type languageResolver struct {
	accountProfile accountProfileResolver
}

// Resolver centralizes request-scoped session, viewer, and language
// resolution behind one explicit package seam.
type Resolver struct {
	auth     authResolver
	viewer   viewerResolver
	language languageResolver
}

const viewerAvatarDeliveryWidthPX = 40

// New builds a resolver from startup dependencies.
func New(deps Dependencies) Resolver {
	accountProfiles := accountProfileResolver{accountClient: deps.AccountClient}
	auth := authResolver{
		sessionClient: deps.SessionClient,
	}
	return Resolver{
		auth: auth,
		viewer: viewerResolver{
			accountProfile: accountProfiles,
			notification:   deps.NotificationClient,
			social:         deps.SocialClient,
			assetBaseURL:   deps.AssetBaseURL,
		},
		language: languageResolver{
			accountProfile: accountProfiles,
		},
	}
}
