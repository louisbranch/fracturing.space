package web

import (
	"context"
	"net/http"
	"sync"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

// PrincipalDependencies carries the clients needed for per-request principal
// resolution (session validation, viewer chrome, language preference, unread
// notifications). These overlap with module clients when the same gRPC
// connection serves both purposes — the separation documents intent.
type PrincipalDependencies struct {
	SessionClient      PrincipalSessionClient
	AccountClient      PrincipalAccountClient
	NotificationClient PrincipalNotificationClient
	SocialClient       PrincipalSocialClient
	AssetBaseURL       string
}

// requestPrincipalState holds per-request cached resolution results.
type requestPrincipalState struct {
	userIDOnce   sync.Once
	userID       string
	viewerOnce   sync.Once
	viewer       module.Viewer
	languageOnce sync.Once
	language     string
}

// requestPrincipalStateKey defines an internal contract used at this web package boundary.
type requestPrincipalStateKey struct{}

// contextFromRequest centralizes this web behavior in one helper seam.
func contextFromRequest(request *http.Request) context.Context {
	if request == nil {
		return context.Background()
	}
	return request.Context()
}
