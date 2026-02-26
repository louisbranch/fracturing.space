package web

import (
	"context"
	"net/http"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	featuresettings "github.com/louisbranch/fracturing.space/internal/services/web/feature/settings"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) appSettingsRouteDependencies(w http.ResponseWriter, r *http.Request) featuresettings.AppSettingsHandlers {
	sess := sessionFromRequest(r, h.sessions)
	return featuresettings.AppSettingsHandlers{
		Authenticate: func(req *http.Request) bool {
			return sessionFromRequest(req, h.sessions) != nil
		},
		RedirectToLogin: func(writer http.ResponseWriter, req *http.Request) {
			http.Redirect(writer, req, routepath.AuthLogin, http.StatusFound)
		},
		HasConnectionsClient: func() bool {
			return h.connectionsClient != nil
		},
		HasCredentialClient: func() bool {
			return h.credentialClient != nil
		},
		ResolveProfileUserID: func(ctx context.Context) (string, error) {
			return h.resolveProfileUserID(ctx, sess)
		},
		GetUserProfile: func(ctx context.Context, req *connectionsv1.GetUserProfileRequest) (*connectionsv1.GetUserProfileResponse, error) {
			return h.connectionsClient.GetUserProfile(ctx, req)
		},
		SetUserProfile: func(ctx context.Context, req *connectionsv1.SetUserProfileRequest) (*connectionsv1.SetUserProfileResponse, error) {
			return h.connectionsClient.SetUserProfile(ctx, req)
		},
		ListCredentials: func(ctx context.Context, req *aiv1.ListCredentialsRequest) (*aiv1.ListCredentialsResponse, error) {
			return h.credentialClient.ListCredentials(ctx, req)
		},
		CreateCredential: func(ctx context.Context, req *aiv1.CreateCredentialRequest) (*aiv1.CreateCredentialResponse, error) {
			return h.credentialClient.CreateCredential(ctx, req)
		},
		RevokeCredential: func(ctx context.Context, req *aiv1.RevokeCredentialRequest) (*aiv1.RevokeCredentialResponse, error) {
			return h.credentialClient.RevokeCredential(ctx, req)
		},
		RenderErrorPage: h.renderErrorPage,
		PageContext: func(req *http.Request) webtemplates.PageContext {
			return h.pageContext(w, req)
		},
	}
}
