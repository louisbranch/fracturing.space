package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

func TestNewHandlerResolvesCookieSessionAtMostOncePerRequest(t *testing.T) {
	t.Parallel()

	auth := newCountingWebAuthClient()
	_, _ = auth.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "user-1"})
	h, err := newTestHandler(Config{
		Dependencies: newDependencyBundle(
			principal.Dependencies{SessionClient: auth, AccountClient: &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}}},
			modules.Dependencies{
				PublicAuth: modules.PublicAuthDependencies{AuthClient: auth},
				Profile:    modules.ProfileDependencies{SocialClient: defaultSocialClient()},
				Settings: modules.SettingsDependencies{
					SocialClient:     defaultSocialClient(),
					AccountClient:    &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}},
					CredentialClient: fakeCredentialClient{},
					AgentClient:      fakeAgentClient{},
				},
			},
		),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if auth.GetWebSessionCalls() != 1 {
		t.Fatalf("GetWebSession calls = %d, want %d", auth.GetWebSessionCalls(), 1)
	}
}
